package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/patrickmn/go-cache"

	"show-live/utils"
)

type Cache struct {
	file string
	*cache.Cache
}

func InitCache(dir string) (*Cache, error) {
	file := path.Join(dir, "cache.json")

	exists, err := utils.PathExists(dir)
	if err != nil {
		return nil, fmt.Errorf("check if dir exists error %v", err)
	}
	var c = cache.New(5*time.Minute, 10*time.Minute)
	if !exists {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("mkdir error %v", err)
		}
	} else {
		jsonItems, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("Error loading cache from file: %s", err)
		}
		items := make(map[string]cache.Item)
		json.Unmarshal(jsonItems, &items)
		for key, item := range items {
			c.Set(key, item.Object, time.Duration(item.Expiration))
		}
	}
	return &Cache{
		file:  file,
		Cache: c,
	}, nil
}

func (c *Cache) SetKey(key, name string, value interface{}) error {
	c.Set(key, value, -1)
	return nil
}

func (c *Cache) Exists(key string) (bool, error) {
	_, ok := c.Get(key)
	if !ok {
		return false, nil
	}
	return true, nil
}

func (c *Cache) GetValue(key string) (interface{}, error) {
	v, ok := c.Get(key)
	if !ok {
		return false, nil
	}
	return v, nil
}

func (c *Cache) GetEventByValue(value int64) ([]string, error) {
	return nil, nil
}

func (c *Cache) Exit() error {
	items := c.Items()
	jsonItems, _ := json.Marshal(items)

	err := ioutil.WriteFile(c.file, jsonItems, 0644)
	if err != nil {
		return fmt.Errorf("Error saving cache to file: %s", err)
	}
	return nil
}
