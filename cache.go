package trafilatura

type Cache struct {
	maxSize int
	keys    []string
	data    map[string]int
}

func NewCache(maxSize int) *Cache {
	return &Cache{
		maxSize: maxSize,
		keys:    []string{},
		data:    make(map[string]int),
	}
}

func (c *Cache) Get(key string) (int, bool) {
	value, exist := c.data[key]
	return value, exist
}

func (c *Cache) Remove(key string) {
	// Check if key exist in cache
	_, exist := c.data[key]
	if !exist {
		return
	}

	// Find that key in list of keys
	var keyIdx int
	for i := 0; i < len(c.keys); i++ {
		if c.keys[i] == key {
			keyIdx = i
			break
		}
	}

	// Remove that key in slice and map
	c.keys = append(c.keys[:keyIdx], c.keys[keyIdx+1:]...)
	delete(c.data, key)
}

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

func (c *Cache) Clear() {
	c.keys = []string{}
	c.data = make(map[string]int)
}
