// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
//
// Copyright (C) 2021 Markus Mobius
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lru

import "slices"

// Cache is a simple implementation for the Least Recently Used (LRU) cache.
type Cache struct {
	maxSize int
	keys    []string
	data    map[string]int
}

// NewCache returns a new Cache with specified max size.
func NewCache(maxSize int) *Cache {
	return &Cache{
		maxSize: maxSize,
		keys:    []string{},
		data:    make(map[string]int),
	}
}

// Get fetch value from the cache.
func (c *Cache) Get(key string) (int, bool) {
	value, exist := c.data[key]
	return value, exist
}

// Remove removes an item from the cache.
func (c *Cache) Remove(key string) {
	// Check if key exist in cache
	_, exist := c.data[key]
	if !exist {
		return
	}

	// Find that key in list of keys
	var keyIdx int
	for i := range c.keys {
		if c.keys[i] == key {
			keyIdx = i
			break
		}
	}

	// Remove that key in slice and map
	c.keys = slices.Delete(c.keys, keyIdx, keyIdx+1)
	delete(c.data, key)
}

// Put stores a given key in the cache.
func (c *Cache) Put(key string, value int) {
	// If key already exist, just put it
	if _, exist := c.data[key]; exist {
		c.data[key] = value
		return
	}

	// If there are no room for new key, remove the oldest
	if len(c.keys) >= c.maxSize && c.maxSize > 0 {
		oldestKey := c.keys[0]
		c.keys = c.keys[1:]
		delete(c.data, oldestKey)
	}

	// Put the new value
	c.keys = append(c.keys, key)
	c.data[key] = value
}

// Clear removes all cache content.
func (c *Cache) Clear() {
	c.keys = []string{}
	c.data = make(map[string]int)
}
