package pamlogix

import (
	"context"
	"database/sql"
	"fmt"
	"net/smtp"

	"github.com/heroiclabs/nakama-common/runtime"
)

type BasePamlogix struct{}

func (b *BasePamlogix) GetType() SystemType {
	return SystemTypeBase
}

func (b *BasePamlogix) GetConfig() any {
	return &BaseSystemConfig{}
}

// RateApp uses the SMTP configuration to receive feedback from players via email.
func (b *BasePamlogix) RateApp(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, username string, score uint32, message string) error {
	// Retrieve SMTP config (replace with your actual config retrieval)
	config := b.GetConfig().(*BaseSystemConfig)
	smtpHost := config.RateAppSmtpAddr
	smtpPort := config.RateAppSmtpPort
	smtpUser := config.RateAppSmtpUsername
	smtpPass := config.RateAppSmtpPassword
	feedbackEmail := config.RateAppSmtpEmailTo

	subject := "App Rating Feedback"
	body := fmt.Sprintf("User: %s (%s)\nScore: %d\nMessage: %s", username, userID, score, message)
	msg := []byte("To: " + feedbackEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)
	if err := smtp.SendMail(addr, auth, smtpUser, []string{feedbackEmail}, msg); err != nil {
		logger.Error("Failed to send feedback email: %v", err)
		return err
	}
	return nil
}

// SetDevicePrefs sets push notification tokens on a user's account so push messages can be received.
func (b *BasePamlogix) SetDevicePrefs(ctx context.Context, db *sql.DB, userID, deviceID, pushTokenAndroid, pushTokenIos string, preferences map[string]bool) error {
	// Implement your logic here
	return nil
}

// Sync processes an operation to update the server with offline state changes.
func (b *BasePamlogix) Sync(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, req *SyncRequest) (*SyncResponse, error) {
	// Implement your logic here
	return &SyncResponse{}, nil
}
