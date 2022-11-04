package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

type Conf struct {
	Tunrnarounds []struct {
		Stand struct {
			Name    string `yaml:"name" json:"name"`
			Cameras []struct {
				Name string `yaml:"name" json:"name"`
				File string `yaml:"file" json:"file"`
			}
		}
		Status string `yaml:"status" json:"status"`
	}
}

func main() {
	var wg sync.WaitGroup

	configFile := flag.String("c", "master-setup.yaml", "config yaml file to use")
	baseUrl := flag.String("baseUrl", "rtmp://master-setup:30900/stream-test", "Base url to stream to")

	flag.Parse()

	fmt.Printf("Staring with %s config.\n", *configFile)

	c, err := readConf(*configFile)
	if err != nil {
		fmt.Println(err)
	}

	for _, turn := range c.Tunrnarounds {
		if turn.Status == "enabled" {

			for _, cam := range turn.Stand.Cameras {
				wg.Add(1)
				go stream(cam.Name, cam.File, *baseUrl, &wg)
			}
		}
	}

	// wg.Add(1)
	// go printTime("Time", &wg)

	fmt.Println("Waiting for streams to finish...")
	wg.Wait()
	fmt.Println("Done!")
}

func printTime(name string, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		fmt.Println(name, " : ", time.Now())
		time.Sleep(1 * time.Second)
	}

}

func readConf(filename string) (*Conf, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	c := &Conf{}

	d := yaml.NewDecoder(file)
	if err := d.Decode(&c); err != nil {
		return nil, err
	}

	return c, nil
}
