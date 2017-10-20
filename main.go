package main

import (
	logstalgia "github.com/SilverCory/go-logstalgia/http"
	logstash "github.com/SilverCory/go-logstash/http"
	"github.com/SilverCory/go-logstash/log"
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
	"strings"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"os"
	"github.com/SilverCory/go-logstalgia/config"
)

func main() {

	conf := &Config{}
	conf.Load()

	logstalgiaInstance := logstalgia.New(&conf.LogstalgiaConfig)
	logstashInstance := logstash.New(conf.AuthKey, log.New())

	logstashInstance.LogCallback = func(path, s string) {
		fmt.Println(path)
		if path != conf.LogFilePathMoitor {
			return
		}

		logentry := &logstalgia.LogEntry{}
		parts := strings.SplitN(s, " | ", 6)
		if len(parts) != 6 {
			fmt.Println( len(parts ))
			return
		}

		logentry.Time = parts[0]
		logentry.Path = parts[1]
		if i, err := strconv.Atoi(parts[2]); err != nil {
			logentry.Size = 1
		} else {
			logentry.Size = i
		}

		logentry.IP = parts[3]
		logentry.Method = parts[4]
		if i, err := strconv.Atoi(parts[5]); err != nil {
			logentry.Result = 418
		} else {
			logentry.Result = i
		}

		fmt.Printf("%#v\n", logentry)
		logstalgiaInstance.Broadcast(logentry)

	}

	router := mux.NewRouter()

	router.HandleFunc("/log/{path:.*}", logstashInstance.HandleLog).Methods("POST")
	router.HandleFunc("/ws", logstalgiaInstance.Socket.UpgradeWebsocket)
	router.HandleFunc("/", logstalgiaInstance.HandleIndex)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	http.Handle("/", router)
	if err := http.ListenAndServe(conf.ListenAddr, nil); err != nil {
		fmt.Println("Fatal err:", err)
	}

}

type Config struct {
	LogstalgiaConfig config.LogstalgiaConfig `json:"logstalgia_config"`
	ListenAddr string `json:"listen_addr"`
	AuthKey string `json:"auth_key"`
	LogFilePathMoitor string `json:"log_file_path_moitor"`
}

func (c *Config) Load() {
	if _, err := os.Stat("./config.json"); os.IsNotExist(err) {
		c.Save()
		fmt.Println("The default configuration has been saved. Please edit this and restart!")
		os.Exit(0)
		return
	} else {
		data, err := ioutil.ReadFile("./config.json")
		if err != nil {
			fmt.Println("There was an error loading the config!", err)
			return
		}

		err = json.Unmarshal(data, c)
		if err != nil {
			fmt.Println("There was an error loading the config!", err)
			os.Exit(1)
			return
		}
	}
}

func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "\t")

	if err != nil {
		fmt.Println("There was an error saving the config!", err)
		return err
	}

	err = ioutil.WriteFile("./config.json", data, 0644)
	if err != nil {
		fmt.Println("There was an error saving the config!", err)
		return err
	}

	return nil

}