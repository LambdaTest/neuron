package emailnotifications

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"time"

	"github.com/LambdaTest/test-at-scale/pkg/errs"
)

func (e *emailNotificationManager) sendBatchEmails(ctx context.Context, batchEmailsData []Message, endpoint string) error {
	multipleMails := &multipleEmails{
		Messages: batchEmailsData,
	}
	reqBody, errM := json.Marshal(multipleMails)
	if errM != nil {
		e.logger.Errorf("failed to marshal reqBody, %v", errM)
		return errM
	}

	if _, errA := e.requests.MakeAPIRequest(ctx, http.MethodPost, endpoint, reqBody, ""); errA != nil {
		e.logger.Errorf("error in sending emails, %v", errA)
		return errA
	}
	return nil
}

func (e *emailNotificationManager) validateAddresses(nameAndAddress map[string]string) ([]string, error) {
	addresses := make([]string, len(nameAndAddress))
	count := 0
	for emailAddress, name := range nameAndAddress {
		if address, err := mail.ParseAddress(emailAddress); err != nil {
			e.logger.Errorf("error in parsing email: %v, error: %v", emailAddress, err)
			return nil, err
		} else {
			addresses[count] = fmt.Sprintf("%s %s", name, address)
		}
		count++
	}
	if len(addresses) == 0 {
		return nil, errs.New("no valid email found")
	}
	return addresses, nil
}

func (e *emailNotificationManager) getCurrentTimeInIST() string {
	loc := time.FixedZone("IST", +5*60*60+30*60)
	now := time.Now().In(loc)
	now = now.Round(time.Millisecond)
	return now.Format(time.RFC1123)
}
