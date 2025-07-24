package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"sync"
	txtTemplate "text/template"

	"github.com/wneessen/go-mail"
)

//go:embed "templates"
var templateFS embed.FS

type Mailer struct {
	dailer       *mail.Client
	sender       string
	initTmpl     sync.Once
	htmlTemplate *template.Template
	txtTemplate  *txtTemplate.Template
}

// Get smtp details from cfg.smtp
func New(host, username, password, sender string, port int) (*Mailer, error) {
	client, err := mail.NewClient(
		host,
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTLSPolicy(mail.TLSMandatory),
		mail.WithPort(port),
	)
	if err != nil {
		return nil, err
	}
	return &Mailer{dailer: client, sender: sender}, nil
}

// Do this fn in background when used as it might block for sending
// also do retrying like 3 times if failed to send
// data rn has activationToken and userID
func (m *Mailer) Send(to string, data any) error {
	// Do once and is cached for later use
	// Im doing thos cause i think paring templates in expensive
	// may be i should have done it in init fn??
	m.initTmpl.Do(m.initializeTemplates)
	msg := mail.NewMsg()
	err := msg.To(to)
	if err != nil {
		return err
	}
	err = msg.From(m.sender)
	if err != nil {
		return err
	}
	subject := new(bytes.Buffer)
	err = m.txtTemplate.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}
	msg.Subject(subject.String())
	// txt := new(bytes.Buffer)
	// err = m.template.ExecuteTemplate(txt, "plainTextBody",data)
	// if err!=nil{
	// 	return err
	// }
	// msg.SetBodyString(mail.TypeTextPlain,txt.String())
	// html:=new(bytes.Buffer)
	// err=m.template.ExecuteTemplate(html,"htmlBody",data)
	// if err!=nil{
	// 	return err
	// }
	// msg.AddAlternativeString(mail.TypeTextHTML,html.String())
	err = msg.SetBodyTextTemplate(m.txtTemplate, data)
	if err != nil {
		return err
	}
	err = msg.AddAlternativeHTMLTemplate(m.htmlTemplate, data)
	if err != nil {
		return err
	}
	return nil
}

// Must be called in the Mialer.initTmpl.Do func
// to do only once on the first call to send func
func (m *Mailer) initializeTemplates() {
	tmpl, err := template.New("email").ParseFS(templateFS, "template/welcome_mail.tmpl")
	if err != nil {
		panic("Couln not parse the email")
	}
	txtTemplate, err := txtTemplate.New("txt").ParseFS(templateFS, "template/welocome_mail_txt.tmpl")
	if err != nil {
		panic("Couln not parse the email")
	}
	m.htmlTemplate = tmpl
	m.txtTemplate = txtTemplate
}
