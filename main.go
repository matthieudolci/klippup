package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/radovskyb/watcher"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	billy "github.com/go-git/go-billy/v5"
	memfs "github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"
	http "github.com/go-git/go-git/v5/plumbing/transport/http"
	memory "github.com/go-git/go-git/v5/storage/memory"
)

const (
	Setup = "setup"
	Run   = "run"
)

type Conf struct {
	Repository string
	Username   string
	Token      string
	Folder     string
}

var storer *memory.Storage
var fs billy.Filesystem

func main() {

	action := os.Args[1]

	run(action)
}

func run(action string) error {

	switch action {
	case Setup:
		return setup()
	case Run:
		e := configExist()
		if e {
			return watch()
		} else {
			return nil
		}
	default:
		return nil
	}
}

func configExist() (exist bool) {
	if _, err := os.Stat("config.yml"); err == nil {
		return true

	} else {
		os.IsNotExist(err)
		log.Error("Config file doesnt exist, please create it first.")
		return false
	}
}

func setup() error {
	repository := os.Args[2]
	username := os.Args[3]
	token := os.Args[4]
	folder := os.Args[5]

	var t = Conf{repository, username, token, folder}

	d, err := yaml.Marshal(&t)
	if err != nil {
		log.WithError(err).Error("Could not generate conf file")
	}

	err = ioutil.WriteFile("config.yml", d, 0777)

	return err
}

func watch() error {

	t := Conf{}

	y, err := ioutil.ReadFile("config.yml")
	if err != nil {
		log.WithError(err).Error("Could not read config file")
	}

	err = yaml.Unmarshal(y, &t)
	if err != nil {
		log.WithError(err).Error("Could not unmarshal yaml file")
	}

	w := watcher.New()

	go func() {
		for {
			select {
			case event := <-w.Event:
				fmt.Println(event)
				commit()
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	err = w.AddRecursive(t.Folder)
	if err != nil {
		log.Fatalln(err)
	}

	err = w.Start(time.Millisecond * 100)
	if err != nil {
		log.Fatalln(err)
	}

	return err
}

func commit() error {
	storer = memory.NewStorage()
	fs = memfs.New()

	t := Conf{}

	y, err := ioutil.ReadFile("config.yml")
	if err != nil {
		log.WithError(err).Error("Could not read config file")
	}

	err = yaml.Unmarshal(y, &t)
	if err != nil {
		log.WithError(err).Error("Could not unmarshal yaml file")
		return err
	}

	auth := &http.BasicAuth{
		Username: t.Username,
		Password: t.Token,
	}

	r, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:  t.Repository,
		Auth: auth,
	})
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	fmt.Println("Repository cloned")

	w, err := r.Worktree()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	_, err = w.Add(t.Folder)
	if err != nil {
		log.WithError(err).Error("Could not add files to commit")
		return err
	}

	_, err = w.Commit("Klipper config file updated!", &git.CommitOptions{
		All: true
	})
	if err != nil {
		log.WithError(err).Error("Could not commit")
		return err
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		log.WithError(err).Error("Could not push to origin")
		return err
	}

	return nil
}
