package config

import (
	"encoding/json"
	"io/ioutil"
	"sync"
	"time"
)

// Structs
type OwnerConfig struct {
	Owner []string `json:"owner"`
}

type WelcomeConfig struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}

type GroupConfig struct {
	Welcome map[string]WelcomeConfig `json:"welcome"`
}

type AFKData struct {
	Reason string    `json:"reason"`
	Time   time.Time `json:"time"`
}

// Global Config
var (
	Owner       OwnerConfig
	Group       GroupConfig
	AFKUsers    = make(map[string]AFKData)
	ConfigMutex sync.RWMutex
)

// Functions
func LoadOwnerConfig() {
	ConfigMutex.Lock()
	defer ConfigMutex.Unlock()
	data, err := ioutil.ReadFile("owner.json")
	if err != nil {
		Owner = OwnerConfig{Owner: []string{"628xxxxxxx8@s.whatsapp.net"}}
		SaveOwnerConfig()
		return
	}
	json.Unmarshal(data, &Owner)
}

func SaveOwnerConfig() {
	data, _ := json.MarshalIndent(Owner, "", "  ")
	ioutil.WriteFile("owner.json", data, 0644)
}

func LoadGroupConfig() {
	data, err := ioutil.ReadFile("groups.json")
	if err == nil {
		json.Unmarshal(data, &Group)
	} else {
		Group.Welcome = make(map[string]WelcomeConfig)
	}
}

func SaveGroupConfig() {
	data, _ := json.MarshalIndent(Group, "", "  ")
	ioutil.WriteFile("groups.json", data, 0644)
}
