package fritzbox

import (
	"fmt"

	"go.uber.org/zap"
)

// See https://avm.de/fileadmin/user_upload/Global/Service/Schnittstellen/AVM_Technical_Note_-_Session_ID.pdf.
type Session struct {
	Challenge string      `xml:"Challenge"` // A challenge provided by the FRITZ!Box.
	SID       string      `xml:"SID"`       // The session id issued by the FRITZ!Box, "0000000000000000" is considered invalid/"no session".
	BlockTime string      `xml:"BlockTime"` // The time that needs to expire before the next login attempt can be made.
	Rights    Permissions `xml:"Rights"`    // The Rights associated withe the session.
}

type Permissions struct {
	Names        []string `xml:"Name"`
	AccessLevels []string `xml:"Access"`
}

// zeroSessionID is the session ID issued by the FRITZ!Box to indicate an
// invalid or "no session".
const zeroSessionID = "0000000000000000"

func (c *Client) login() error {
	err := c.getXML(&c.session, "/login_sid.lua", "sid", c.session.SID)
	if err != nil {
		return fmt.Errorf("failed to get login challenge: %w", err)
	}

	if c.session.SID != zeroSessionID {
		return nil // session is still valid
	}

	c.logger.Debug("Authenticating new session at FRITZ!Box API", zap.String("base_url", c.BaseURL.String()))
	challengeResponse := c.session.solveChallenge(c.Password)
	err = c.getXML(&c.session, "/login_sid.lua",
		"response", challengeResponse,
		"username", c.Username,
	)
	if err != nil {
		return fmt.Errorf("failed to submit challenge response: %w", err)
	}

	if c.session.SID == "" || c.session.SID == zeroSessionID {
		return fmt.Errorf("failed to solve authentication challenge, check username and password")
	}

	return nil
}

func (s Session) solveChallenge(password string) string {
	challengeAndPassword := s.Challenge + "-" + password
	return s.Challenge + "-" + toUTF16andMD5(challengeAndPassword)
}

func (c *Client) logout() error {
	if c.session.SID == "" {
		return nil // we don't have a session
	}

	c.logger.Debug("Logging out from FRITZ!Box API")
	_, err := c.get("/login_sid.lua", "sid", c.session.SID, "logout", "true")
	return err
}
