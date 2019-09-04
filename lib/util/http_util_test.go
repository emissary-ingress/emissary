package util

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const greeting = "Hello, client!."

func simpleTs(t *testing.T, greeting string) (ts *httptest.Server) {
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Responding")
		fmt.Fprint(w, greeting)
	}))
	return
}

func simpleTs404(t *testing.T, greeting string) (ts *httptest.Server) {
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Responding 404")
		w.WriteHeader(404)
		fmt.Fprint(w, greeting)
	}))
	return
}

var client = &SimpleClient{&http.Client{}}

func TestGetBodyBytesSimpleOK(t *testing.T) {
	ts := simpleTs(t, greeting)
	defer ts.Close()

	t.Logf("Reading body bytes")
	data, err := client.GetBodyBytes(ts.URL)
	if err != nil {
		t.Error("Get body bytes sucks", err)
	}
	actual := string(data)
	if actual != greeting {
		t.Errorf("returned value sucks expected <%s> actual <%s>", greeting, actual)
	}
}
func TestGetBodyBytesSimple404(t *testing.T) {
	ts := simpleTs404(t, greeting)
	defer ts.Close()

	t.Logf("Reading body bytes")
	data, err := client.GetBodyBytes(ts.URL)
	if err != nil {
		actual := string(data)
		if actual != greeting {
			t.Errorf("returned value sucks expected <%s> actual <%s>", greeting, actual)
		}
		return
	}
	t.Error("Get body bytes sucks", err)
}

func TestGetBodyBytesSimpleNoCheck(t *testing.T) {
	ts := simpleTs404(t, greeting)
	defer ts.Close()

	t.Logf("Reading body bytes")
	data, err := client.GetBodyBytes(ts.URL, nil)
	if err != nil {
		t.Error("Get body bytes sucks", err)
	}
	actual := string(data)
	if actual != greeting {
		t.Errorf("returned value sucks expected <%s> actual <%s>", greeting, actual)
	}
}

func pretend404OK(response *http.Response, data []byte) error {
	if response.StatusCode == 404 {
		return nil
	}
	return fmt.Errorf("Weird code %d, expected 404", response.StatusCode)
}

func TestGetBodyBytesSimplePretend404OK(t *testing.T) {
	ts := simpleTs404(t, greeting)
	defer ts.Close()

	t.Logf("Reading body bytes")
	data, err := client.GetBodyBytes(ts.URL, pretend404OK)
	if err != nil {
		t.Error("Get body bytes sucks", err)
	}
	actual := string(data)
	if actual != greeting {
		t.Errorf("returned value sucks expected <%s> actual <%s>", greeting, actual)
	}
}

type sampleData struct {
	Foo string
	Bar string
}

func TestGetBodyJSONOK(t *testing.T) {
	jsonString := `{"Foo":"haha","Bar":"hoho"}`
	ts := simpleTs(t, jsonString)
	defer ts.Close()

	s := sampleData{}
	t.Logf("Reading body json")
	err := client.GetBodyJSON(ts.URL, &s)
	if err != nil {
		t.Error("Get body json sucks", err)
	}
	if s.Foo != "haha" || s.Bar != "hoho" {
		t.Errorf("returned value sucks expected <%s> actual <%s>", jsonString, s)
	}

}
