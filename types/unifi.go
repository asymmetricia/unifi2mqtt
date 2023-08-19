package types

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/unpoller/unifi"
)

type Client struct {
	Name, Mac string
	Present   bool
}

type Unifi struct {
	Username      string
	Password      string
	Host          string
	Port          int
	VerifyTls     bool
	DeviceTimeout time.Duration
	Matchers      []*Matcher
	seenClients   map[string]string
	unifiClient   *unifi.Unifi
	ch            chan Client
	loginBackoff  time.Duration
}

func (u *Unifi) Login(log zerolog.Logger) error {
	time.Sleep(u.loginBackoff)

	var err error
	u.unifiClient, err = unifi.NewUnifi(&unifi.Config{
		User:      u.Username,
		Pass:      u.Password,
		URL:       fmt.Sprintf("https://%s:%d", u.Host, u.Port),
		VerifySSL: u.VerifyTls,
		ErrorLog: func(msg string, fmt ...interface{}) {
			log.Error().Msgf(msg, fmt...)
		},
		DebugLog: func(msg string, fmt ...interface{}) {
			log.Debug().Msgf(msg, fmt...)
		},
	})

	if err == nil {
		u.loginBackoff = 0
	} else if u.loginBackoff == 0 {
		u.loginBackoff = time.Second
	} else if u.loginBackoff < time.Minute {
		u.loginBackoff *= 2
	}

	return err
}

func (u *Unifi) Chan() <-chan Client {
	return u.ch
}

func (u *Unifi) Start(log zerolog.Logger) error {
	if u.unifiClient == nil {
		if err := u.Login(log); err != nil {
			return fmt.Errorf("logging in: %w", err)
		}
	}

	clients, err := u.clients(log)
	if err != nil {
		return fmt.Errorf("getting initial list of clients: %w", err)
	}

	go func(clients []Client, u *Unifi) {
		ticker := time.NewTicker(time.Minute)
		for {
			for _, client := range clients {
				u.ch <- client
			}

			<-ticker.C

			clients, err = u.clients(log)
			if err != nil {
				log.Error().Err(err).Msg("could not get clients, but will retry")
			}

		}
	}(clients, u)

	return nil
}

func NewUnifi(username, password, host string, port int, verifyTls bool,
	deviceTimeout time.Duration, matchers []*Matcher, log zerolog.Logger,
) (*Unifi, error) {
	ret := &Unifi{
		Username:      username,
		Password:      password,
		Host:          host,
		Port:          port,
		VerifyTls:     verifyTls,
		DeviceTimeout: deviceTimeout,
		Matchers:      matchers,
		seenClients:   map[string]string{},
		ch:            make(chan Client),
	}

	if err := ret.Start(log); err != nil {
		return nil, err
	}

	return ret, nil
}

func (u *Unifi) clients(log zerolog.Logger) ([]Client, error) {
	sites, err := u.unifiClient.GetSites()
	if err != nil && strings.Contains(err.Error(), "code from server 401") {
		err = u.Login(log)
		if err == nil {
			sites, err = u.unifiClient.GetSites()
		}
	}
	if err != nil {
		return nil, fmt.Errorf("getting sites: %w", err)
	}

	unifiClients, err := u.unifiClient.GetClients(sites)
	if err != nil && strings.Contains(err.Error(), "code from server 401") {
		err = u.Login(log)
		if err == nil {
			unifiClients, err = u.unifiClient.GetClients(sites)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("getting clients: %w", err)
	}

	offline := map[string]string{}
	online := map[string]string{}
clientLoop:
	for _, client := range unifiClients {
		for _, matcher := range u.Matchers {
			if matcher.Match(client.Name) {
				seen := time.Unix(client.LastSeen.Int64(), 0)
				log.Debug().Msgf("found client %s, MAC %s, Last Seen %s",
					client.Name, client.Mac, seen)
				if time.Since(seen) < u.DeviceTimeout {
					online[client.Name] = client.Mac
				} else {
					offline[client.Name] = client.Mac
				}
				continue clientLoop
			}
		}
	}

	// Any device that was previously seen on-line but that is now no longer
	// found is off-line.
	for name, mac := range u.seenClients {
		if _, ok := online[name]; !ok {
			offline[name] = mac
		}
	}
	u.seenClients = online

	var ret []Client
	for name, mac := range offline {
		ret = append(ret, Client{name, mac, false})
	}
	for name, mac := range online {
		ret = append(ret, Client{name, mac, true})
	}

	return ret, nil
}
