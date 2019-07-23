package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testStruct struct {
	Name  string
	Value int
	Uint  uint

	Dura time.Duration
	//Time   time.Time `fmt:"2006-01-02"`
	Enable bool

	Float32 float32 `flag:"float32"`
	Float64 float64 `flag:"float64"`
}

func TestGoConfig_Load(t *testing.T) {

	c := testStruct{
		Dura:    time.Second,
		Value:   1,
		Uint:    2,
		Float32: 12.3,
		Enable:  true,
	}

	// setup environment vars
	os.Setenv("DURA", "12s")
	os.Setenv("VALUE", "8")
	os.Setenv("FLOAT_64", "123.4")
	os.Setenv("TIME", "2019-05-06")
	os.Setenv("NAME", "env")

	os.Args = []string{"go-config", "-c=test/test.toml", "-name=flag", "-enable=false", "-float32=55", "-dura=5s"}

	if err := New(&c).Load(); err != nil {
		t.Fatal(err)
	}
	exp := testStruct{
		Name: "flag",
		//		Time:    trial.TimeDay("2012-02-04"),
		Dura:    5 * time.Second,
		Value:   10,
		Uint:    2,
		Float32: 55,
		Float64: 123.4,
	}

	assert.Equal(t, c, exp)

}

func TestLoadEnv(t *testing.T) {
	c := testStruct{
		Dura:    time.Second,
		Value:   1,
		Uint:    2,
		Float32: 12.3,
	}

	// test the environment loading
	os.Setenv("DURA", "12s")
	os.Setenv("VALUE", "8")
	os.Setenv("FLOAT_64", "123.4")
	os.Setenv("TIME", "2019-05-06")
	os.Setenv("NAME", "env")
	if err := LoadEnv(&c); err != nil {
		t.Fatal("environment load error ", err)
	}
	exp := testStruct{
		Name: "env",
		Dura: 12 * time.Second,
		//	Time:    trial.TimeDay("2019-05-06"),
		Value:   8,
		Uint:    2,
		Float32: 12.3,
		Float64: 123.4,
	}
	assert.Equal(t, c, exp)
}

func TestLoadFile(t *testing.T) {
	c := testStruct{
		Dura:    time.Second,
		Value:   1,
		Uint:    2,
		Float32: 12.3,
	}

	if err := LoadFile("test/test.toml", &c); err != nil {
		t.Fatal("toml file load error: ", err)
	}
	exp := testStruct{
		Dura: time.Second,
		//		Time:    trial.TimeDay("2010-08-10"),
		Value:   10,
		Uint:    2,
		Float32: 99.9,
		Enable:  true,
		Name:    "toml",
	}
	assert.Equal(t, c, exp)
}

func TestLoadFlag(t *testing.T) {
	c := testStruct{
		Dura:    time.Second,
		Value:   1,
		Uint:    2,
		Float32: 12.3,
	}

	os.Args = []string{"go-config", "-name=flag", "-enable=false", "-float32=55", "-dura=5s"}
	if err := LoadFlag(&c); err != nil {
		t.Fatal("flag load error ", err)
	}
	exp := testStruct{
		Name: "flag",
		//		Time:    trial.TimeDay("2012-02-04"),
		Dura:    5 * time.Second,
		Value:   1,
		Uint:    2,
		Float32: 55,
	}
	assert.Equal(t, c, exp)
}
