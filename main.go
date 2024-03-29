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
	"time"
	"math/rand"
)

var charmap map[rune]rune
var acceptable = []rune{'0', '1', '2','3','4','5','6','7','8','9', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z'}

func init() {
	rand.Seed(time.Now().UnixNano())
	charmap = make(map[rune]rune)
	go func() {
		for _, v := range acceptable {
			charmap[v] = acceptable[rand.Intn(len(acceptable))]
		}
		time.Sleep(time.Minute*30)
	}()
}

func main() {

	conf := &Config{}
	conf.Load()
	
	var d = func(f string, i ...interface{}) {
		fmt.Printf(f + "\n", i...)	
	}

	d("Starting instance")
	logstalgiaInstance := logstalgia.New(&conf.LogstalgiaConfig)
	logstashInstance := logstash.New(conf.AuthKey, log.New())

	logstashInstance.LogCallback = func(path, s string) {
		if path != conf.LogFilePathMoitor {
			return
		}

		logentry := &logstalgia.LogEntry{}
		parts := strings.SplitN(s, " | ", -1)
		if len(parts) != 7 {
			return
		}

		logentry.Time = parts[1]
		logentry.Path = parts[2]
		if i, err := strconv.Atoi(parts[3]); err != nil {
			logentry.Size = 1
		} else {
			logentry.Size = i
		}

		logentry.IP = hideIP(parts[4])
		logentry.Method = parts[5]
		if i, err := strconv.Atoi(parts[6]); err != nil {
			logentry.Result = 418
		} else {
			logentry.Result = i
		}

		d("%#v", logentry)
		logstalgiaInstance.Broadcast(logentry)

	}

	router := mux.NewRouter()

	router.HandleFunc("/log/{path:.*}", logstashInstance.HandleLog).Methods("POST")
	router.HandleFunc("/ws", logstalgiaInstance.Socket.UpgradeWebsocket)
	router.HandleFunc("/", logstalgiaInstance.HandleIndex)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	d("Starting server")
	http.Handle("/", router)
	if err := http.ListenAndServe(conf.ListenAddr, nil); err != nil {
		panic(err)
	}

}

type Config struct {
	LogstalgiaConfig config.LogstalgiaConfig `json:"logstalgia_config"`
	ListenAddr string `json:"listen_addr"`
	AuthKey string `json:"auth_key"`
	LogFilePathMoitor string `json:"log_file_path_moitor"`
	Debug bool `json:"debug"`
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

func hideIP(s string) (r string) {
	s = strings.ToUpper(strings.TrimSpace(s))
	parts := strings.Split(s, ".")
	if len(parts) == 4 {
		return hide(parts, ".")
	}

	parts = strings.Split(s, ":")
	if len(parts) >= 2{
		return hide(parts, ":")
	}

	return mapChars(s)
}

func hide(i []string, delimiter string) string {
	i[3] = mapChars(i[3])
	i[2] = mapChars(i[2])
	return strings.Join(i, delimiter)
}

func mapChars(i string) (r string) {
	for _, v := range i {
		r += string(charmap[v])
	}
	return

}
