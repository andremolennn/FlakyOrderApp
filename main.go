package main

import (
	"fmt"
	"net/http"
	"sync"
)

var (
	//username and password
	users = map[string]string{
		"user1": "password123",
	}

	//balance
	balances = map[string]float64{
		"user1": 1,
	}

	//List of product and price
	products = map[string]float64{
		"apple":  1.0,
		"banana": 0.5,
	}
	cart = make(map[string]map[string]int)
	mu   = &sync.Mutex{}
)

func main() {
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/add_to_cart/", addToCartHandler)
	http.HandleFunc("/checkout", checkoutHandler)
	http.ListenAndServe(":8080", nil)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		//valid username & password
		if storedPass, ok := users[username]; ok && storedPass == password {
			http.SetCookie(w, &http.Cookie{
				Name:  "session",
				Value: username,
			})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		//invalid username or password
		http.Error(w, "Wrong User or Password", http.StatusUnauthorized)
		return
	}

	// show form login
	fmt.Fprint(w, `
        <h1>Login Page</h1>
        <form action="/login" method="post">
            <label for="username">Username:</label>
            <input type="text" id="username" name="username"><br>
            <label for="password">Password:</label>
            <input type="password" id="password" name="password"><br>
            <input type="submit" value="Login">
        </form> 
    `)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	cookie, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := cookie.Value
	userCart, ok := cart[username]
	if !ok {
		userCart = make(map[string]int)
		cart[username] = userCart
	}

	// Show User and Balance
	balance := balances[username]
	fmt.Fprintf(w, "Hello, %s!<br> Your balance: $%.2f<br><br>", username, balance)

	//list of product available
	fmt.Fprintf(w, "Available products:<br>")
	for product, price := range products {
		// Button add product to cart
		fmt.Fprintf(w, "%s: $%.2f <a href=\"/add_to_cart/%s\">Add to cart</a><br> ", product, price, product)
	}

	//list of cart
	fmt.Fprint(w, "<br>Your cart:<br>")
	for product, quantity := range userCart {
		fmt.Fprintf(w, "%s: %d<br>", product, quantity)
	}

	//checkout button
	fmt.Fprint(w, `<a href="/checkout">Checkout</a>`)
}

func addToCartHandler(w http.ResponseWriter, r *http.Request) {
	product := r.URL.Path[len("/add_to_cart/"):]
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	username := cookie.Value
	userCart, ok := cart[username]
	if !ok {
		userCart = make(map[string]int)
		cart[username] = userCart
	}

	// Check if product exists or not
	if _, ok := products[product]; !ok {
		http.Error(w, "Product not available", http.StatusNotFound)
		return
	}

	userCart[product]++
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func checkoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	username := cookie.Value
	userCart, ok := cart[username]
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	total := 0.0
	for product, quantity := range userCart {
		price, ok := products[product]
		if !ok {
			continue
		}
		total += price * float64(quantity)
	}

	//Not enough balance (total > balance)
	if total > balances[username] {
		http.Error(w, "Not Enough Balance! Top up your balance.", http.StatusPaymentRequired)
		return
	}

	//updating balance
	balances[username] -= total

	// Clear the user's cart after checkout
	delete(cart, username)

	// Print success message along with total and remaining balance
	remainingBalance := balances[username]
	fmt.Fprintf(w, "Checkout successful! Total: $%.2f, <br> Remaining balance: $%.2f<br><br>", total, remainingBalance)

	// Provide link back to main menu
	fmt.Fprintf(w, `<a href="/">Finish and Back to Main Menu</a>`)
}
