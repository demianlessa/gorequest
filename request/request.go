package request

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"
)

var instance *Request

type auth struct {
	Username string
	Password string
	Bearer   string
}

// TODO: Add constructor for Options ?
type Option struct {
	Url     string
	Headers map[string]string
	Auth    *auth
	Body    interface{}
	JSON    interface{}
}

type Request struct {
	client  *http.Client
	Timeout time.Duration
}

func NewAuth(username, password, bearer string) *auth {
	return &auth{
		Username: username,
		Password: password,
		Bearer:   bearer,
	}
}

func New() *Request {
	r := new(Request)

	r.Timeout = 30 * time.Second

	r.client = &http.Client{
		Timeout: r.Timeout,
	}

	return r
}

func NewRequest(url string) (*http.Response, []byte, error) {
	o := &Option{
		Url: url,
	}

	return Get(o)
}

func (r *Request) Post(o *Option) (*http.Response, []byte, error) {
	return r.doRequest("POST", o)
}

func Post(o *Option) (*http.Response, []byte, error) {
	return getInstance().Post(o)
}

func (r *Request) Put(o *Option) (*http.Response, []byte, error) {
	return r.doRequest("PUT", o)
}

func Put(o *Option) (*http.Response, []byte, error) {
	return getInstance().Put(o)
}

func (r *Request) Get(o *Option) (*http.Response, []byte, error) {
	// REMARKS: For the time being, the Body of a GET request will be ignored. For more information, read below or refer to the HTTP Specification.
	// REMARKS: There is a lot of ambiguity to suggest that most servers won't inspect the body of a GET request. Clients like Postman disable the Body tab when performing a GET request.
	// Ref: https://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html.
	// Section 4.3: A message-body MUST NOT be included in a request if the specification of the request method (section 5.1.1) does not allow sending an entity-body in requests.
	// Section 5.2: The exact resource identified by an Internet request is determined by examining both the Request-URI and the Host header field.
	// Section 9.3: The GET method means retrieve whatever information (in the form of an entity) is identified by the Request-URI.
	o.Body = nil

	return r.doRequest("GET", o)
}

func Get(o *Option) (*http.Response, []byte, error) {
	return getInstance().Get(o)
}

func (r *Request) Delete(o *Option) (*http.Response, []byte, error) {
	// REMARKS: Ignore Body - RFC2616
	o.Body = nil

	return r.doRequest("DELETE", o)
}

func Delete(o *Option) (*http.Response, []byte, error) {
	return getInstance().Delete(o)
}

// ********** Private methods/functions **********
// REMARKS: Used internally by non-instance methods
func getInstance() *Request {
	if instance == nil {
		instance = New()
	}

	return instance
}

// REMARKS: The user/pwd can be provided in the URL when doing Basic Authentication (RFC 1738)
func splitUserNamePassword(u string) (usr, pwd string, err error) {
	reg, err := regexp.Compile("^(http|https|mailto)://")

	if err != nil {
		return "", "", err
	}

	s := reg.ReplaceAllString(u, "")

	if reg, err := regexp.Compile("@(.+)"); err != nil {
		return "", "", err
	} else {
		v := reg.ReplaceAllString(s, "")

		c := strings.Split(v, ":")

		if len(c) < 1 {
			return "", "", errors.New("No credentials found in URI")
		}

		return c[0], c[1], nil
	}
}

// REMARKS: Returns a buffer with the body of the request - Content-Type header is set accordingly
func getRequestBody(o *Option) *bytes.Buffer {
	j := reflect.Indirect(reflect.ValueOf(o.JSON))

	if j.Kind() == reflect.String || j.Kind() == reflect.Struct {
		o.Body = o.JSON
		o.JSON = true
		j = reflect.Indirect(reflect.ValueOf(o.JSON))
	}

	b := reflect.Indirect(reflect.ValueOf(o.Body))

	buff := make([]byte, 0)
	body := new(bytes.Buffer)
	contentType := ""

	switch b.Kind() {
	case reflect.String:
		// REMARKS: This takes care of a JSON serialized string
		buff = []byte(b.String())
		body = bytes.NewBuffer(buff)

		// TODO: Need to set headers accordingly (Other headers other than the two below ?
		if j.Bool() {
			contentType = "application/json"
		} else {
			contentType = "text/plain"
		}
		break
	case reflect.Struct:
		if j.Bool() {
			if buff, err := json.Marshal(b.Interface()); err != nil {
				panic(err)
			} else {
				body = bytes.NewBuffer(buff)
			}

			contentType = "application/json"
		} else if err := binary.Write(body, binary.BigEndian, b); err != nil {
			// TODO: Test to ensure that we can safely serialize the body
			panic(err)
		}
		break
	}

	// TODO: Change headers property to be a struct ?
	o.Headers["Content-Type"] = contentType

	return body
}

// REMARKS: The Body in the http.Response will be closed when returning a response to the caller
func (r *Request) doRequest(m string, o *Option) (*http.Response, []byte, error) {
	if o.Headers == nil {
		o.Headers = make(map[string]string)
	}
	body := getRequestBody(o)
	req, err := http.NewRequest(m, o.Url, body)

	if err != nil {
		panic(err)
	}

	if o.Auth != nil {
		if o.Auth.Bearer != "" {
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", o.Auth.Bearer))
		} else if o.Auth.Username != "" && o.Auth.Password != "" {
			req.SetBasicAuth(o.Auth.Username, o.Auth.Password)
		}
	} else if usr, pwd, err := splitUserNamePassword(o.Url); err != nil {
		// TODO: Should we panic if an error is returned or silently ignore this - maybe give some warning ?
		//panic(err)
	} else {
		if usr != "" && pwd != "" {
			req.SetBasicAuth(usr, pwd)
		}
	}

	// TODO: Validate headers against known list of headers ?
	// TODO: Ensure headers are only set once
	// TODO: If JSON property set, add Content-Type: application/json if not already set in o.Headers
	for k, v := range o.Headers {
		req.Header.Add(k, v)
	}

	resp, err := r.client.Do(req)

	defer resp.Body.Close()

	if err != nil {
		return resp, nil, err
	}

	if body, err := ioutil.ReadAll(resp.Body); err != nil {
		return resp, nil, err
	} else {
		return resp, body, nil
	}
}
