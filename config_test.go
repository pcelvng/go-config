package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/jbsmith7741/trial"
)

type testStruct struct {
	Name  string
	Value int
	Uint  uint

	Dura   time.Duration
	Time   time.Time `format:"2006-01-02"`
	Enable bool

	Float32 float32 `flag:"float32"`
	Float64 float64 `flag:"float64"`
	Pointer *childStruct
}

type childStruct struct {
	Count  *int
	Amount *float64
}

func TestGoConfig_Load(t *testing.T) {
	type input struct {
		config testStruct
		envs   map[string]string
		flags  []string
	}
	fn := func(v ...interface{}) (interface{}, error) {
		in := v[0].(input)
		if in.envs == nil {
			in.envs = make(map[string]string)
		}
		if in.flags == nil {
			in.flags = make([]string, 0)
		}
		for k, v := range in.envs {
			if err := os.Setenv(k, v); err != nil {
				return nil, err
			}
		}
		defer func() {
			for k := range in.envs {
				os.Setenv(k, "")
			}
			// reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		}()

		os.Args = append([]string{"go-config"}, in.flags...)
		err := New(&in.config).Load()
		return in.config, err
	}
	cases := trial.Cases{
		"default": {
			Input: input{
				config: testStruct{
					Dura:    time.Second,
					Value:   1,
					Uint:    2,
					Float32: 12.3,
					Enable:  true,
				}},
			Expected: testStruct{
				Dura:    time.Second,
				Value:   1,
				Uint:    2,
				Float32: 12.3,
				Enable:  true,
			},
		},
		"file": {
			Input: input{
				config: testStruct{
					Dura:    time.Second,
					Value:   1,
					Uint:    2,
					Float32: 12.3,
				},
				flags: []string{"-c=test/test.toml"},
			},
			Expected: testStruct{
				Name:    "toml",
				Time:    trial.TimeDay("2010-08-10"),
				Dura:    10 * time.Second,
				Enable:  true,
				Value:   10,
				Uint:    2,
				Float32: 99.9,
			},
		},
		"flag": {
			Input: input{
				config: testStruct{
					Dura:    time.Second,
					Value:   1,
					Uint:    2,
					Float32: 12.3,
					Enable:  true,
				},
				flags: []string{"-time=2012-02-04", "-name=flag", "-enable=false", "-float32=55", "-dura=5s"},
			},
			Expected: testStruct{
				Name:    "flag",
				Time:    trial.TimeDay("2012-02-04"),
				Dura:    5 * time.Second,
				Value:   1,
				Uint:    2,
				Float32: 55,
			},
		},
		"env": {
			Input: input{
				config: testStruct{
					Dura:    time.Second,
					Value:   1,
					Uint:    2,
					Float32: 12.3,
				},
				envs: map[string]string{
					"DURA":     "12s",
					"VALUE":    "8",
					"FLOAT_64": "123.4",
					"TIME":     "2019-05-06",
					"NAME":     "env"},
			},
			Expected: testStruct{
				Name:    "env",
				Time:    trial.TimeDay("2019-05-06"),
				Dura:    12 * time.Second,
				Value:   8,
				Uint:    2,
				Float32: 12.3,
				Float64: 123.4,
			},
		},
		"env+file": {
			Input: input{
				config: testStruct{
					Dura:    time.Second,
					Value:   1,
					Uint:    2,
					Float32: 12.3,
				},
				envs: map[string]string{
					"DURA":     "12s",
					"VALUE":    "8",
					"FLOAT_64": "123.4",
					"TIME":     "2019-05-06",
					"NAME":     "env"},
				flags: []string{"-c=test/test.toml"},
			},
			Expected: testStruct{
				Name:    "toml",
				Time:    trial.TimeDay("2010-08-10"),
				Dura:    10 * time.Second,
				Value:   10,
				Uint:    2,
				Float32: 99.9,
				Float64: 123.4,
				Enable:  true,
			},
		},
		"env+file+flag": {
			Input: input{
				config: testStruct{
					Dura:    time.Second,
					Value:   1,
					Uint:    2,
					Float32: 12.3,
					Enable:  true,
				},
				envs: map[string]string{
					"DURA":     "12s",
					"VALUE":    "8",
					"FLOAT_64": "123.4",
					"TIME":     "2019-05-06",
					"NAME":     "env"},
				flags: []string{"-c=test/test.toml", "-time=2012-02-04", "-name=flag", "-enable=false", "-float32=55", "-dura=5s"},
			},
			Expected: testStruct{
				Name:    "flag",
				Time:    trial.TimeDay("2012-02-04"),
				Dura:    5 * time.Second,
				Value:   10,
				Uint:    2,
				Float32: 55,
				Float64: 123.4,
			},
		},
	}
	trial.New(fn, cases).SubTest(t)

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
		Name:    "env",
		Dura:    12 * time.Second,
		Time:    trial.TimeDay("2019-05-06"),
		Value:   8,
		Uint:    2,
		Float32: 12.3,
		Float64: 123.4,
	}
	if eq, diff := trial.Equal(c, exp); !eq {
		t.Error(diff)
	}
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
		Dura:    10 * time.Second,
		Time:    trial.TimeDay("2010-08-10"),
		Value:   10,
		Uint:    2,
		Float32: 99.9,
		Enable:  true,
		Name:    "toml",
	}
	if eq, s := trial.Equal(c, exp); !eq {
		t.Error(s)
	}
}

func TestLoadFlag(t *testing.T) {
	c := testStruct{
		Dura:    time.Second,
		Value:   1,
		Uint:    2,
		Float32: 12.3,
	}
	defer func() {
		// clean up flag state so later tests can re-register flags
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		defaultCfg = New(nil)
	}()

	os.Args = []string{"go-config", "-name=flag", "-enable=false", "-float32=55", "-dura=5s", "-time=2012-02-04"}
	if err := LoadFlag(&c); err != nil {
		t.Fatal("flag load error ", err)
	}
	exp := testStruct{
		Name:    "flag",
		Time:    trial.TimeDay("2012-02-04"),
		Dura:    5 * time.Second,
		Value:   1,
		Uint:    2,
		Float32: 55,
	}
	if eq, diff := trial.Equal(c, exp); !eq {
		t.Error(diff)
	}
}

func TestArgs(t *testing.T) {
	type input struct {
		args []string
	}
	type output struct {
		Args []string
		Name string
	}
	fn := func(v ...interface{}) (interface{}, error) {
		in := v[0].(input)
		defer func() {
			// clean up flag state so later tests can re-register flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			defaultCfg = New(nil)
		}()

		c := &testStruct{}
		os.Args = append([]string{"go-config"}, in.args...)
		os.Unsetenv("NAME")
		if err := Load(c); err != nil {
			return nil, err
		}
		return output{Args: Args(), Name: c.Name}, nil
	}
	cases := trial.Cases{
		"no positional args": {
			Input:    input{args: []string{"-name=foo"}},
			Expected: output{Args: []string{}, Name: "foo"},
		},
		"positional after flags": {
			Input:    input{args: []string{"-name=foo", "file1", "file2"}},
			Expected: output{Args: []string{"file1", "file2"}, Name: "foo"},
		},
		"positional before flags": {
			Input:    input{args: []string{"file1", "file2", "-name=foo"}},
			Expected: output{Args: []string{"file1", "file2"}, Name: "foo"},
		},
		"positional before and after flags": {
			Input:    input{args: []string{"before", "-name=foo", "after"}},
			Expected: output{Args: []string{"before", "after"}, Name: "foo"},
		},
		"interleaved flags and positionals": {
			Input:    input{args: []string{"a", "-name=foo", "b", "-enable=true", "c"}},
			Expected: output{Args: []string{"a", "b", "c"}, Name: "foo"},
		},
		"only positional": {
			Input:    input{args: []string{"a", "b"}},
			Expected: output{Args: []string{"a", "b"}},
		},
		"space-separated flag value among positionals": {
			Input:    input{args: []string{"pos1", "-name", "foo", "pos2"}},
			Expected: output{Args: []string{"pos1", "pos2"}, Name: "foo"},
		},
	}
	trial.New(fn, cases).SubTest(t)
}

func TestArgs_beforeLoad(t *testing.T) {
	defaultCfg = New(nil)
	got := Args()
	if got != nil {
		t.Errorf("Args() got %v want nil", got)
	}
}

func TestGoConfig_ConfigPath(t *testing.T) {
	type input struct {
		configPath string
		flags      []string
	}
	fn := func(v ...interface{}) (interface{}, error) {
		in := v[0].(input)
		defer func() {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		}()

		c := testStruct{
			Dura:    time.Second,
			Value:   1,
			Uint:    2,
			Float32: 12.3,
		}
		os.Args = append([]string{"go-config"}, in.flags...)
		err := New(&c).ConfigPath(in.configPath).Disable(OptEnv | OptEnvFile).Load()
		return c, err
	}
	cases := trial.Cases{
		"loads default path when -c not provided": {
			Input: input{configPath: "test/test.toml"},
			Expected: testStruct{
				Name:    "toml",
				Time:    trial.TimeDay("2010-08-10"),
				Dura:    10 * time.Second,
				Enable:  true,
				Value:   10,
				Uint:    2,
				Float32: 99.9,
			},
		},
		"-c overrides default path": {
			Input: input{
				configPath: "test/test.toml",
				flags:      []string{"-c=test/test.yaml"},
			},
			Expected: testStruct{
				Name:    "yaml",
				Time:    trial.TimeDay("2010-08-10"),
				Dura:    10 * time.Second,
				Enable:  true,
				Value:   10,
				Uint:    2,
				Float32: 12.3,
			},
		},
		"empty default does not load file": {
			Input: input{configPath: ""},
			Expected: testStruct{
				Dura:    time.Second,
				Value:   1,
				Uint:    2,
				Float32: 12.3,
			},
		},
		"missing default path returns error": {
			Input:     input{configPath: "missing.toml"},
			ShouldErr: true,
		},
	}
	trial.New(fn, cases).SubTest(t)
}

func TestOptions(t *testing.T) {
	opt := defaultOpts
	opt &^= OptToml | OptFlag | OptFiles
	if opt != 0b11000011 {
		t.Errorf("Expected binary value of 11000011 got %b", opt)
	}

	if opt.isEnabled(OptToml) {
		t.Error("toml should be disabled")
	}
	if !opt.isEnabled(OptEnv) {
		t.Error("env should be enabled")
	}
	if !opt.isEnabled(OptEnvFile) {
		t.Error("envfile should be enabled")
	}

	// verify double disable
	v := &goConfig{options: defaultOpts}
	v.Disable(OptFlag)
	if v.options.isEnabled(OptFlag) {
		t.Error("Flag should be disabled")
	}
	v.Disable(OptFlag)
	if v.options.isEnabled(OptFlag) {
		t.Error("Flag 2nd should stay disabled")
	}
	if v.options != 0b11011111 {
		t.Errorf("Expected only flag bit off %b!=%b", 0b11011111, v.options)
	}
}
