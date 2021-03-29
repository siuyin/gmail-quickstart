package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	ms := newMailService()
	var em *email
	//em = &email{From: "me", To: "siuyin@beyondbroadcast.com", Subject: "testing", Body: "<h1>Hello</h1><p>This is some text."}
	em = emailWithAttachment()
	ms.send(em)
	//ms.listLabels()
}

func emailWithAttachment() *email {
	em := &email{
		From:        "me",
		To:          "siuyin@beyondbroadcast.com",
		Subject:     "test email with multiple attachments",
		Body:        "<h1>Greetings</h1><p>Please see attached files.",
		Attachments: []string{"test.csv", "/h/junk.txt"},
	}
	return em
}

type mailService struct {
	srv *gmail.Service
}

func newMailService() *mailService {
	return &mailService{srv: newGmailService()}
}

func (m mailService) listLabels() {
	listLabels(m.srv)
}

func (m mailService) send(email *email) {
	user := "me"
	msg := &gmail.Message{Raw: email.base64URL()}
	out, err := m.srv.Users.Messages.Send(user, msg).Do()
	if err != nil {
		log.Printf("send: %v", err)
	}
	fmt.Println(out)
}

func newGmailService() *gmail.Service {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, gmail.GmailSendScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}
	return srv
}

func listLabels(srv *gmail.Service) {
	user := "me"
	r, err := srv.Users.Labels.List(user).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve labels: %v", err)
	}
	if len(r.Labels) == 0 {
		fmt.Println("No labels found.")
		return
	}
	fmt.Println("Labels:")
	for _, l := range r.Labels {
		fmt.Printf("- %s\n", l.Name)
	}
}

type email struct {
	From, To, Subject string
	Body              string
	Attachments       []string
}

func newEmail(from, to, subj, body string, attachedFilenames ...string) *email {
	return &email{
		From:        from,
		To:          to,
		Subject:     subj,
		Body:        body,
		Attachments: attachedFilenames,
	}
}

func (m email) String() string {
	if len(m.Attachments) == 0 {
		return fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n", m.From, m.To, m.Subject, m.Body)
	}
	return m.multipart()
}

func (m email) multipart() string {
	b := &bytes.Buffer{}
	mw := multipart.NewWriter(b)
	fmt.Fprintf(b, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n",
		m.From, m.To, m.Subject, mw.Boundary())

	w, err := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(w, "%s\r\n", m.Body)

	for _, v := range m.Attachments {
		f, err := os.Open(v)
		if err != nil {
			log.Fatalf("could not open attachment: %s", v)
		}
		defer f.Close()

		mpf, err := mw.CreateFormFile(filepath.Base(v), filepath.Base(v))
		if err != nil {
			log.Fatalf("could not create form file: %s", v)
		}

		_, err = io.Copy(mpf, f)
		if err != nil {
			log.Fatalf("could not copy attachment: %s", v)
		}
	}
	mw.Close()
	return b.String()
}

// see also https://play.golang.org/p/k1SCRLH9EMe for multipart email generation
// and this for multipart/mixed https://play.golang.org/p/Ifztb4dKFW2

func (m email) base64URL() string {
	return base64.URLEncoding.EncodeToString([]byte(m.String()))
}
