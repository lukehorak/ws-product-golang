package main

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Struct to store user IP and last request time
type user struct {
	limiter     *rate.Limiter
	lastRequest time.Time
}

var (
	users = make(map[string]*user)
	mu    sync.Mutex
)

func init() {
	go clearUsers()
}

/*///////////////////////////////////////////
	getUser()
		description:
			Getter function for getting a user's limiter
			if they exist, or initializing a new user if
			they don't
		parameters:
			ip [string]:
				The user's IP address, used as the key in
				the users map
		return value:
			u.limiter [*rate.Limiter]: the limiter for
			the user with given IP 'ip'
///////////////////////////////////////////*/

func getUser(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	u, exists := users[ip]
	if !exists {
		limiter := rate.NewLimiter(1, 3)
		users[ip] = &user{limiter, time.Now()}
		return limiter
	}
	u.lastRequest = time.Now()
	return u.limiter
}

/*///////////////////////////////////////////
	clearUsers()
		description:
			Clears map of user IPs 3 minutes after last request (to avoid an
			infinite-length of user IPs). Called in a go routine in init(),
			running alongside the limiter
		parameters:
			none!
		return value:
			nil
///////////////////////////////////////////*/
func clearUsers() {
	for {
		time.Sleep(time.Minute)
		mu.Lock()
		defer mu.Unlock()

		for ip, u := range users {
			if time.Now().Sub(u.lastRequest) > 3*time.Minute {
				delete(users, ip)
			}
		}
	}
}

func rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Server Error! Whoopsie daisies", http.StatusInternalServerError)
			return
		}

		limiter := getUser(ip)
		if limiter.Allow() == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
