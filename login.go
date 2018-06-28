package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
)

// Target is a struct to hold the original target requested by a user
type Target struct {
	Referer string
}

// cookie handling

var cookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

func getUserName(request *http.Request) (userName string) {
	if cookie, err := request.Cookie("session"); err == nil {
		cookieValue := make(map[string]string)
		if err = cookieHandler.Decode("session", cookie.Value, &cookieValue); err == nil {
			userName = cookieValue["name"]
		}
	}
	return userName
}

func setSession(userName string, response http.ResponseWriter) {
	value := map[string]string{
		"name": userName,
	}
	if encoded, err := cookieHandler.Encode("session", value); err == nil {
		cookie := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(response, cookie)
	}
}

func clearSession(response http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(response, cookie)
}

// authenticate handler
func authenticate(name string, pass string) (isAuthenticated bool) {
	isAuthenticated = false

	// TODO:  Move url to a properties file
	url := "http://openam.example.com:8080/openam/json/realms/root/authenticate"
	fmt.Println("URL:>", url)

	req, err := http.NewRequest("POST", url, nil)
	req.Header.Set("X-OpenAM-Username", name)
	req.Header.Set("X-OpenAM-Password", pass)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

	if resp.StatusCode == 200 {
		isAuthenticated = true
	}
	return isAuthenticated
}

// login handler

func loginHandler(response http.ResponseWriter, request *http.Request) {

	redirectTarget := "/"

	name := request.FormValue("name")
	pass := request.FormValue("password")

	fmt.Println("LoginHandler - Request: ", request)
	fmt.Println("LoginHandler - Referer: ", request.Header.Get("Referer"))

	a := request.Header.Get("Referer")

	b, err := url.ParseQuery(a)

	if err != nil {
		fmt.Println("oh shit: ", b)
	} else {
		for key, value := range b {
			fmt.Println(key)
			redirectTarget = value[0]
			fmt.Println("New Redirect Target: ", redirectTarget)
		}
	}

	request.ParseForm()

	// parse request and get referer

	// redirect to the target extracted from referer with the AM session cookie

	// if name and password fields aren't null ...

	if name != "" && pass != "" {
		// .. check credentials ..
		isAuthenticated := authenticate(name, pass)

		if isAuthenticated {
			setSession(name, response)
			//redirectTarget = "/internal"
		} else {
			redirectTarget = "/"
		}
	}
	http.Redirect(response, request, redirectTarget, 302)
}

// logout handler

func logoutHandler(response http.ResponseWriter, request *http.Request) {
	clearSession(response)
	http.Redirect(response, request, "/", 302)
}

const indexPage = `
 <h1>Login</h1>
 <hr>
 <form method="post" action="/login">
	<label for="name">User name: </label>
	<input type="text" id="name" name="name"><br>
	<label for="password">Password: </label>
	<input type="password" id="password" name="password"><br>
	<hr>
	<button type="submit">Login</button>
 </form>
`

func indexPageHandler(response http.ResponseWriter, request *http.Request) {

	if err := request.ParseForm(); err != nil {
		fmt.Printf("Here is the Request: ", request)
		log.Printf("Error parsing form: %s", err)
		return
	}

	fmt.Fprintf(response, indexPage)

}

const internalPage = `
<h1>Internal</h1>
<hr>
<small>User: %s</small>
<form method="post" action="/logout">
	<button type="submit">Logout</button>
</form>
`

func internalPageHandler(response http.ResponseWriter, request *http.Request) {
	userName := getUserName(request)
	if userName != "" {
		fmt.Fprintf(response, internalPage, userName)
	} else {
		http.Redirect(response, request, "/", 302)
	}
}

// server main method

var router = mux.NewRouter()

func main() {
	router.HandleFunc("/", indexPageHandler)
	router.HandleFunc("/internal", internalPageHandler)

	router.HandleFunc("/login", loginHandler).Methods("POST")
	router.HandleFunc("/logout", logoutHandler).Methods("POST")

	http.Handle("/", router)
	http.ListenAndServe(":9090", nil)
}
