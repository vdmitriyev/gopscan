package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"github.com/wneessen/go-mail"
)

const DirEmailOutbox = "emails"

type ConfigEmailType struct {
	EmailServer        string `env:"EMAIL_SERVER" env-default:"127.0.0.1"`
	EmailPort          int    `env:"EMAIL_PORT" env-default:"587"`
	EmailUseTLS        bool   `env:"EMAIL_USE_TLS" env-default:true`
	EmailUser          string `env:"EMAIL_USERNAME" env-default:"admin"`
	EmailPassword      string `env:"EMAIL_PASSWORD" env-default:"admin"`
	EmailSender        string `env:"EMAIL_SENDER" env-default:"admin@localhost"`
	EmailAdminNotifier string `env:"EMAIL_ADMIN_NOTIFIER" env-default:"admin@localhost"`
	EmailReadonlyMode  bool   `env:"EMAIL_READONLY_MODE" env-default:true`
}

var ConfigEmail ConfigEmailType

func createDirectory(dirPath string) {
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dirPath, 0755)
		if err != nil && !os.IsExist(err) {
			log.Printf("error creating directory: %s\n", err)
		} else if err == nil {
			log.Printf("directory created successfully: %s\n", dirPath)
		}
	}
}

func ReadConfigs() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := cleanenv.ReadEnv(&ConfigEmail); err != nil {
		log.Println("was not able to read env")
	}
}

func SendEmail(subject, content string) error {
	ReadConfigs()
	toEmail := ConfigEmail.EmailAdminNotifier
	readonly := ConfigEmail.EmailReadonlyMode
	m := mail.NewMsg()
	if err := m.From(ConfigEmail.EmailSender); err != nil {
		return fmt.Errorf("failed to set From address: %s", err)
	}

	if err := m.To(toEmail); err != nil {
		return fmt.Errorf("failed to set To address: %s", err)
	}

	m.Subject(subject)
	m.SetBodyString(mail.TypeTextPlain, content)

	if !readonly {

		newEmail, err := mail.NewClient(
			ConfigEmail.EmailServer,
			mail.WithPort(ConfigEmail.EmailPort),
			mail.WithSMTPAuth(mail.SMTPAuthLogin),
			mail.WithUsername(ConfigEmail.EmailUser),
			mail.WithPassword(ConfigEmail.EmailPassword),
			//mail.WithDebugLog(),
		)

		if err != nil {
			return fmt.Errorf("failed to create mail client: %s", err)
		}

		if err := newEmail.DialAndSend(m); err != nil {
			return fmt.Errorf("failed to send mail: %s", err)
		}

		return saveEmailToFile(m, toEmail)
	} else {
		return saveEmailToFile(m, toEmail)
	}
}

func saveEmailToFile(m *mail.Msg, toEmail string) error {
	createDirectory(DirEmailOutbox)
	newUUID7, _ := uuid.NewV7()
	fileName := fmt.Sprintf("%s_report_%s.txt", newUUID7.String(), toEmail)
	filePath := filepath.Join(DirEmailOutbox, fileName)

	if err := m.WriteToFile(filePath); err != nil {
		return fmt.Errorf("failed to save mail to file: %s, %s", filePath, err)
	}

	return nil
}
