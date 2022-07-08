package cache

import (
	"sync"
	"bankservice/user"
	"errors"
	"os"
	"io"
	"encoding/csv"
	"strconv"
)

var (
	AlreadyExistsErr = errors.New("user with that id already exists")
	NotExistsErr = errors.New("user with that id not exists")
	emptyUser = user.User{}
)

type Cache struct {
	Users map[string]user.User
	sync.RWMutex
}

func New() *Cache {
	c := &Cache{}
	c.Users = make(map[string]user.User)
	return c
}

func (c *Cache) Get(id string) (*user.User, error){
	c.RLock()
	defer c.RUnlock()
	if c.Users[id] == emptyUser {
		return &emptyUser, NotExistsErr
	}
	user := c.Users[id]
	return &user, nil
}

func (c *Cache) Add(u *user.User) error{
	c.Lock()
	defer c.Unlock()
	if c.Users[u.Id] == emptyUser {
		c.Users[u.Id] = *u
		return nil
	}
	return AlreadyExistsErr
}

func (c *Cache) Delete(id string) error{
	c.Lock()
	defer c.Unlock()
	if c.Users[id] == emptyUser {
		return NotExistsErr
	}
	delete(c.Users, id)
	return nil
}

func (c *Cache) ChangeBalance(id string, sum float64) error {
	c.Lock()
	defer c.Unlock()
	user := c.Users[id]
	if user == emptyUser {
		return NotExistsErr
	}
	user.Balance = user.Balance+sum
	c.Users[id] = user
	return nil
}

func (c *Cache) RestoreFromFile(fileName string) error {
	file, err := os.OpenFile(fileName, os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	r := csv.NewReader(file)
	r.Comma = ';'

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		balance, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			return err
		}
		c.Add(&user.User{Id: record[0], AuthKey: record[1], Balance: balance})
	}
	os.Truncate(fileName, 0)
	return nil
}

func (c *Cache) ScreenToFile(fileName string) error {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	w := csv.NewWriter(file)
	defer w.Flush()
	w.Comma = ';'
	for _, v := range c.Users {
		err = w.Write([]string{v.Id, v.AuthKey, strconv.FormatFloat(v.Balance, 'f', -1, 64)})
		if err != nil {
			return err
		}
	}
	return nil
}



