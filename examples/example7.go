package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"regexp"

	"github.com/seborama/govcr"
)

const example7CassetteName = "MyCassette7"

// Order is out example body we want to modify.
type Order struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Example7 is an example use of govcr.
// This will show how bodies can be rewritten.
// We will take a varying ID from the request URL, neutralize it and also change the ID in the body of the response.
func Example7() {
	cfg := govcr.VCRConfig{
		Logging: true,
	}

	// Regex to extract the ID from the URL.
	reOrderID := regexp.MustCompile(`/order/([^/]+)`)

	// Create a local test server that serves out responses.
	handler := func(w http.ResponseWriter, r *http.Request) {
		id := reOrderID.FindStringSubmatch(r.URL.String())
		if len(id) < 2 {
			w.WriteHeader(404)
			return
		}

		w.WriteHeader(200)
		b, err := json.Marshal(Order{
			ID:   id[1],
			Name: "Test Order",
		})
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(b)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// The filter will neutralize a value in the URL.
	// In this case we rewrite /order/{random} to /order/1234
	// and replacing the host so it doesn't depend on the random port number.
	replacePath := govcr.RequestFilter(func(req govcr.Request) govcr.Request {
		req.URL.Path = "/order/1234"
		req.URL.Host = "127.0.0.1"
		return req
	})

	// Only execute when we match path.
	cfg.RequestFilters.Add(replacePath.OnPath(`/order/`))

	cfg.ResponseFilters.Add(
		govcr.ResponseFilter(func(resp govcr.Response) govcr.Response {
			req := resp.Request()

			// Find the requested ID:
			orderID := reOrderID.FindStringSubmatch(req.URL.String())

			// Unmarshal body.
			var o Order
			err := json.Unmarshal(resp.Body, &o)
			if err != nil {
				panic(err)
			}

			// Change the ID
			o.ID = orderID[1]

			// Replace the body.
			resp.Body, err = json.Marshal(o)
			if err != nil {
				panic(err)
			}
			return resp
		}).OnStatus(200),
	)

	orderID := fmt.Sprint(rand.Int63())
	vcr := govcr.NewVCR(example7CassetteName, &cfg)

	// create a request with our custom header and a random url part.
	req, err := http.NewRequest("GET", server.URL+"/order/"+orderID, nil)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("GET", req.URL.String())

	// run the request
	resp, err := vcr.Client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	// print outcome.
	fmt.Println("Status code:", resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Returned Body:", string(body))
	fmt.Printf("%+v\n", vcr.Stats())
}
