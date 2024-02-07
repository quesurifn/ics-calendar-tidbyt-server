package main

var appConfig = struct {
	AppName string

	Port string
	Env  string
	JWT  struct {
		PublicKey string
	}
}{
	AppName: "Tidbyt ICS Server",
	Port:    "8080",
}
