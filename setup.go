package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fgrosse/fritz-mon/fritzbox"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

func runSetup() {
	input := bufio.NewReader(os.Stdin)
	ask := func(question, defaultVal string) string {
		text := "> " + question
		if defaultVal != "" {
			text += " [" + defaultVal + "]"
		}

		fmt.Print(text + " : ")
		line, err := input.ReadString('\n')
		if err != nil {
			fmt.Printf("ERROR: Failed to read user input: %v\n", err)
			os.Exit(1)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			return defaultVal
		}

		return line
	}

	fmt.Println("~~ FRITZ!Box Monitor Setup ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
	fmt.Println("The following setup will help you to create a configuration file so fritz-mon")
	fmt.Println("can access your FRITZ!Box. The questions in brackets show you the default value.")
	fmt.Println("You can abort the setup at any point via ctrl+c without any side effects.")
	fmt.Println()

	configPath := ask("Where do you want to store your configuration file?", "fritz-mon.yml")
	fmt.Println("  Checking if a configuration file already exists at this location... ")
	if strings.HasPrefix(configPath, "~/") {
		configPath = filepath.Join(os.Getenv("HOME"), configPath[2:])
	}

	configPath, _ = filepath.Abs(configPath)
	conf, err := LoadConfiguration(configPath, zap.NewNop())
	switch {
	case errors.Is(err, os.ErrNotExist):
		fmt.Printf("  ✔ No file found at %q\n", configPath)
		f, err := os.Create(configPath)
		f.Close()
		if err != nil {
			fmt.Println("  ✘ Seems like we cannot write to that location")
			fmt.Println("    " + err.Error())
			os.Exit(1)
		}

	default:
		fmt.Printf("  ✘ There is already a file at %q\n", configPath)
		if err != nil {
			fmt.Println("  ✘ The file cannot be loaded as configuration due to the following error:")
			fmt.Println("    " + err.Error())
		} else {
			fmt.Println("  ✔ The existing config file is valid")
		}

		answer := ask("Do you want to overwrite this file?", "no")
		if strings.ToLower(answer) != "yes" && strings.ToLower(answer) != "y" {
			fmt.Println("  Aborting setup. Have a nice day!")
			os.Exit(1)
		}
	}

listenAddrStep:
	listenAddr := ask("At which address should fritz-mon open its HTTP server?", conf.ListenAddr)
	_, err = url.Parse("http://" + listenAddr)
	if err != nil {
		fmt.Println("  ✘ This is not a valid address. Please use the HOST:PORT notation (e.g. localhost:1234)")
		goto listenAddrStep
	}

	fmt.Println("  Checking if we can use this address to open an HTTP server... ")
	server := &http.Server{Addr: listenAddr, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ping" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})}
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.ListenAndServe()
	}()
	select {
	case err := <-errChan:
		fmt.Printf("  ✘ There was an error opening the HTTP server at %q\n", listenAddr)
		fmt.Println("    " + err.Error())
		goto listenAddrStep

	case <-time.After(time.Second):
		resp, err := http.Get("http://" + listenAddr + "/ping")
		if err != nil {
			fmt.Printf("  ✘ There was an error sending HTTP requests to the server")
			fmt.Println("    " + err.Error())
			goto listenAddrStep
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("  ✘ The server responded with an unexpected status code: %s", resp.Status)
			goto listenAddrStep
		}
	}

	_ = server.Close()
	fmt.Println("  ✔ The listen address is valid and can be used")
	conf.ListenAddr = listenAddr

intervalStep:
	answer := ask("At which interval should fritz-mon request metrics from the FRITZ!Box API?", conf.DeviceMonitoringInterval.String())
	fmt.Println("  Checking provided interval value... ")
	interval, err := time.ParseDuration(answer)
	if err != nil {
		fmt.Println(`  ✘ Invalid interval. Please use a duration such as "5m" for five minutes or 30s for thirty seconds.`)
		fmt.Println("    " + err.Error())
		goto intervalStep
	}

	if interval < 10*time.Second {
		fmt.Printf("  ✘ The interval %q is too short. Please choose a duration of at least 10 seconds.\n", interval)
		fmt.Println("    Typically one minute or more is more than enough.")
		goto intervalStep
	}

	fmt.Println("  ✔ The interval is valid and can be used")
	conf.DeviceMonitoringInterval = interval

baseURLStep:
	baseURL := ask("What is the URL of your FRITZ!Box", conf.FritzBox.BaseURL)
	u, err := url.Parse(baseURL)
	if err != nil {
		fmt.Println("  ✘ This is not a valid URL:")
		fmt.Println("    " + err.Error())
		goto baseURLStep
	}

	if u.Scheme == "https" {
		fmt.Println("  ✘ Connecting via HTTPS to your FRITZ!Box is not yet supported")
		fmt.Println("    Please try again with http instead")
		goto baseURLStep
	}

	conf.FritzBox.BaseURL = baseURL

usernameStep:
	conf.FritzBox.Username = ask("What is the name of the FRITZ!Box that fritz-mon should use", conf.FritzBox.Username)
	if conf.FritzBox.Username == "" {
		fmt.Println("  ✘ The username cannot be empty and there is no sensible default")
		goto usernameStep
	}

	conf.FritzBox.Password = ask("What is the password for this user? Please remember that passwords are stored in plaintext and will be shown here when you are typing", "")

	fmt.Println("  Checking connection to FRITZ!Box by listing connected SmartHome devices... ")
	client, err := fritzbox.New(conf.FritzBox.BaseURL, conf.FritzBox.Username, conf.FritzBox.Password, zap.NewNop())
	if err != nil {
		fmt.Println("  ✘ Failed to create FRITZ!Box client")
		fmt.Println("    " + err.Error())
		os.Exit(1)
	}

	devices, err := client.Devices()
	if err != nil {
		fmt.Println("  ✘ Failed to list devices")
		fmt.Println("    " + err.Error())
	} else {
		fmt.Printf("  ✔ connection to FRITZ!Box API is working (found %d SmartHome devices)\n", len(devices))
	}

	fmt.Println("  Running final checks on configuration...")
	err = conf.Validate()
	if err != nil {
		fmt.Println("  ✘ Issue found:")
		fmt.Println("    " + err.Error())
	}

	f, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println("Writing configuration file")
		fmt.Println("  ✘ Failed to open file for writing")
		fmt.Println("    " + err.Error())
		os.Exit(1)
	}

	err = yaml.NewEncoder(f).Encode(conf)
	if err != nil {
		fmt.Println("Writing configuration file")
		fmt.Println("  ✘ Failed to write config file")
		fmt.Println("    " + err.Error())
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Printf("Your configuration file has been saved to %q\n", configPath)
	fmt.Println("")
	fmt.Println("You can edit that file manually now at any time.")
	fmt.Println("fritz-mon only reads it once when it starts so if you want your")
	fmt.Println("changes to take effect you have to restart the program.")
	fmt.Println("")
	fmt.Println("You can start fritz-mon with this command:")
	fmt.Println("")
	fmt.Printf("  fritz-mon -config=%s\n", configPath)
	fmt.Println("")
	fmt.Println("Please also review permissions to the config file if you are on a multi-user system!")
}
