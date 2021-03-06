package mailer

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	yaml "gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	flag.String("smtp-path", "../../configs/smtp_test.yaml", "Set SMTP path.")
	flag.Parse()

	log.SetOutput(ioutil.Discard)
	logrus.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestNotification(t *testing.T) {
	var smtpServer smtpServer
	// The code below inits the SMTP configuration for sending emails
	// The path of the yaml config file of test smtp server
	file, err := os.Open(flag.Lookup("smtp-path").Value.(flag.Getter).Get().(string))
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return
	}
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&smtpServer)
	if err != nil {
		log.Printf("Mailer: unexpected error executing command: %v", err)
		return
	}

	contentData := CommonContentData{}
	contentData.CommonData.Tenant = "test"
	contentData.CommonData.Username = "johndoe"
	contentData.CommonData.Name = "John"
	contentData.CommonData.Email = []string{"john.doe@edge-net.org"}

	multiProviderData := MultiProviderData{}
	multiProviderData.Name = "test"
	multiProviderData.Host = "12.12.123.123"
	multiProviderData.Status = "Status"
	multiProviderData.Message = []string{"Status Message"}
	multiProviderData.CommonData = contentData.CommonData

	resourceAllocationData := ResourceAllocationData{}
	resourceAllocationData.Name = "test"
	resourceAllocationData.OwnerNamespace = "tenant-test"
	resourceAllocationData.ChildNamespace = "tenant-test-namespace-test"
	resourceAllocationData.Tenant = "test"
	resourceAllocationData.CommonData = contentData.CommonData

	verifyContentData := VerifyContentData{}
	verifyContentData.Code = "verificationcode"
	verifyContentData.CommonData = contentData.CommonData

	createKubeconfig := func(contentData interface{}, done chan bool) {
		registrationData := contentData.(CommonContentData)
		// Creating temp config file to be consumed by setUserRegistrationContent()
		var file, err = os.Create(fmt.Sprintf("%s/assets/kubeconfigs/%s-%s.cfg", dir, registrationData.CommonData.Tenant,
			registrationData.CommonData.Username))
		if err != nil {
			t.Errorf("Failed to create temp %s/assets/kubeconfigs/%s-%s.cfg file", dir, registrationData.CommonData.Tenant,
				registrationData.CommonData.Username)
		}
		<-done
		file.Close()
		os.Remove(fmt.Sprintf("%s/assets/kubeconfigs/%s-%s.cfg", dir, registrationData.CommonData.Tenant,
			registrationData.CommonData.Username))
	}

	cases := map[string]struct {
		Content  interface{}
		Expected []string
	}{
		"user-email-verification":                    {verifyContentData, []string{verifyContentData.CommonData.Tenant, verifyContentData.CommonData.Username, verifyContentData.Code}},
		"user-email-verification-update":             {verifyContentData, []string{verifyContentData.CommonData.Tenant, verifyContentData.CommonData.Username, verifyContentData.Code}},
		"user-email-verified-alert":                  {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-email-verified-notification":           {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-registration-successful":               {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"tenant-email-verification":                  {verifyContentData, []string{verifyContentData.CommonData.Tenant, verifyContentData.CommonData.Username, verifyContentData.CommonData.Name, verifyContentData.Code}},
		"tenant-email-verified-alert":                {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"tenant-creation-successful":                 {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username}},
		"acceptable-use-policy-accepted":             {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username}},
		"acceptable-use-policy-renewal":              {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"acceptable-use-policy-expired":              {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"node-contribution-successful":               {multiProviderData, []string{multiProviderData.CommonData.Tenant, multiProviderData.CommonData.Username, multiProviderData.CommonData.Name, multiProviderData.Name, multiProviderData.Host, multiProviderData.Message[0]}},
		"node-contribution-failure":                  {multiProviderData, []string{multiProviderData.CommonData.Tenant, multiProviderData.CommonData.Username, multiProviderData.CommonData.Name, multiProviderData.Name, multiProviderData.Host, multiProviderData.Message[0]}},
		"node-contribution-failure-support":          {multiProviderData, []string{multiProviderData.CommonData.Tenant, multiProviderData.Name, multiProviderData.Host, multiProviderData.Message[0]}},
		"tenant-validation-failure-name":             {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"tenant-validation-failure-email":            {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"tenant-email-verification-malfunction":      {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username}},
		"tenant-creation-failure":                    {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"tenant-email-verification-dubious":          {contentData, []string{contentData.CommonData.Tenant}},
		"user-validation-failure-name":               {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-validation-failure-email":              {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-email-verification-malfunction":        {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username}},
		"user-creation-failure":                      {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-cert-failure":                          {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-kubeconfig-failure":                    {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username, contentData.CommonData.Name}},
		"user-email-verification-dubious":            {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username}},
		"user-email-verification-update-malfunction": {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username}},
		"user-deactivation-failure":                  {contentData, []string{contentData.CommonData.Tenant, contentData.CommonData.Username}},
	}

	for k, tc := range cases {
		t.Run(fmt.Sprintf("%s", k), func(t *testing.T) {
			if k == "user-registration-successful" {
				done := make(chan bool)
				go createKubeconfig(tc.Content, done)
				defer func() { done <- true }()
				time.Sleep(500 * time.Millisecond)
			}

			t.Run("template", func(t *testing.T) {
				_, body := prepareNotification(k, tc.Content, smtpServer)
				bodyString := body.String()
				for _, expected := range tc.Expected {
					if !strings.Contains(bodyString, expected) {
						t.Errorf("Email template %v.html failed. Template malformed. Expected \"%v\" in the template not found\n", k, expected)
					}
				}
			})

			/*t.Run("send", func(t *testing.T) {
				err = Send(k, tc.Content)
				util.OK(t, err)
				time.Sleep(200 * time.Millisecond)
			})*/
		})
	}
}
