package cache

import (
	"reflect"
	"testing"
	"bankservice/user"
	"github.com/stretchr/testify/assert"
)

func ExampleUser1() *user.User {
	user := user.User{Id:"user1", AuthKey:"authkey1", Balance:100.0}
	return &user
}

func ExampleUser2() *user.User {
	user := user.User{Id:"user2", AuthKey:"authkey2", Balance:200.0}
	return &user
}

func ExampleUser3() *user.User {
	user := user.User{Id:"user3", AuthKey:"authkey3", Balance:300.0}
	return &user
}

func ExampleCache1() *Cache {
	c := New()
	c.Users["user1"] = user.User{Id:"user1", AuthKey:"authkey1", Balance:100.0}
	c.Users["user2"] = user.User{Id:"user2", AuthKey:"authkey2", Balance:200.0}
	return c
}

func ExampleCache2() *Cache {
	c := New()
	c.Users["user1"] = user.User{Id:"user1", AuthKey:"authkey1", Balance:100.0}
	c.Users["user2"] = user.User{Id:"user2", AuthKey:"authkey2", Balance:200.0}
	c.Users["user3"] = user.User{Id:"user3", AuthKey:"authkey3", Balance:300.0}
	return c
}

func Test_New(t *testing.T) {
	assert := assert.New(t)
	want := &Cache{}
	want.Users = make(map[string]user.User)

	got := New()
	assert.Equal(want, got, "they should be equal")
}

func Test_Get(t *testing.T) {
	assert := assert.New(t)
	c := ExampleCache1()

	got, err := c.Get("user1")
	assert.Nil(err)
	want := ExampleUser1()
	assert.Equal(want, got, "they should be equal")

	got, err = c.Get("blablabla")
	assert.NotNil(err)
	assert.NotEqual(want, got, "they should be not equal")
}

func Test_Add(t *testing.T) {
	assert := assert.New(t)
	c := ExampleCache1()

	err := c.Add(ExampleUser1())
	assert.NotNil(err)

	err = c.Add(ExampleUser3())
	assert.Nil(err)
	assert.Equal(ExampleCache2(), c, "they should be equal")
}

func Test_Delete(t *testing.T) {
	assert := assert.New(t)
	c := ExampleCache2()

	err := c.Delete("user3")
	assert.Nil(err)
	assert.Equal(ExampleCache1(), c, "they should be equal")

	err = c.Delete("blabla")
	assert.NotNil(err)
	assert.Equal(ExampleCache1(), c, "they should be equal")
}

func Test_ChangeBalance(t *testing.T) {
	assert := assert.New(t)
	c := ExampleCache1()

	err := c.ChangeBalance("user1", -200.0)
	assert.NotNil(err)

	err = c.ChangeBalance("user1", -100.0)
	assert.Nil(err)

	err = c.ChangeBalance("user1", 100.0)
	assert.Nil(err)

	err = c.ChangeBalance("blabla", 100.0)
	assert.NotNil(err)
}

func Test_ScreenToFile(t *testing.T) {
	assert := assert.New(t)
	c := ExampleCache1()

	err := c.ScreenToFile("test_cache_got.csv")
	assert.Nil(err)

	err = c.ScreenToFile("blablabla.csv")
	assert.NotNil(err)
}

func Test_RestoreFromFile(t *testing.T) {
	assert := assert.New(t)
	got := New()
	want := ExampleCache1()

	err := got.RestoreFromFile("blablabla.csv")
	assert.NotNil(err)
	assert.NotEqual(reflect.DeepEqual(got, want), true, "they should be not equal")

	err = got.RestoreFromFile("test_cache_want.csv")
	assert.Nil(err)
	assert.Equal(reflect.DeepEqual(got, want), true, "they should be equal")
}