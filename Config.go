package main

type Config struct {
	Listen  string            `json:"listen"`
	Servers map[string]string `json:"servers"`
}
