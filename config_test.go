package config

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hydronica/trial"
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
	fn := func(in input) (testStruct, error) {
		if in.envs == nil {
			in.envs = make(map[string]string)
		}
		if in.flags == nil {
			in.flags = make([]string, 0)
		}
		for k, v := range in.envs {
			if err := os.Setenv(k, v); err != nil {
				return testStruct{}, err
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
	cases := trial.Cases[input, testStruct]{
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
	fn := func(in input) (output, error) {
		defer func() {
			// clean up flag state so later tests can re-register flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			defaultCfg = New(nil)
		}()

		c := &testStruct{}
		os.Args = append([]string{"go-config"}, in.args...)
		os.Unsetenv("NAME")
		if err := Load(c); err != nil {
			return output{}, err
		}
		return output{Args: Args(), Name: c.Name}, nil
	}
	cases := trial.Cases[input, output]{
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
		configPaths []string
		flags       []string
	}
	fn := func(in input) (testStruct, error) {
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
		err := New(&c).ConfigPath(in.configPaths...).Disable(OptEnv | OptEnvFile).Load()
		return c, err
	}
	cases := trial.Cases[input, testStruct]{
		"loads default path when -c not provided": {
			Input: input{configPaths: []string{"test/test.toml"}},
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
				configPaths: []string{"test/test.toml"},
				flags:       []string{"-c=test/test.yaml"},
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
			Input: input{configPaths: []string{""}},
			Expected: testStruct{
				Dura:    time.Second,
				Value:   1,
				Uint:    2,
				Float32: 12.3,
			},
		},
		"missing default path warns and continues": {
			Input: input{configPaths: []string{"missing.toml"}},
			Expected: testStruct{
				Dura:    time.Second,
				Value:   1,
				Uint:    2,
				Float32: 12.3,
			},
		},
		"first of several defaults wins": {
			Input: input{configPaths: []string{"test/test.toml", "test/test.yaml"}},
			Expected: testStruct{
				Name:    "toml",
				Time:    trial.Day("2010-08-10"),
				Dura:    10 * time.Second,
				Enable:  true,
				Value:   10,
				Uint:    2,
				Float32: 99.9,
			},
		},
		"falls through to second when first missing": {
			Input: input{configPaths: []string{"missing.toml", "test/test.yaml"}},
			Expected: testStruct{
				Name:    "yaml",
				Time:    trial.Day("2010-08-10"),
				Dura:    10 * time.Second,
				Enable:  true,
				Value:   10,
				Uint:    2,
				Float32: 12.3,
			},
		},
		// Directory paths succeed os.Stat but are not loadable; skip them and keep trying.
		"falls through to second when first is a directory": {
			Input: input{configPaths: []string{"test", "test/test.yaml"}},
			Expected: testStruct{
				Name:    "yaml",
				Time:    trial.Day("2010-08-10"),
				Dura:    10 * time.Second,
				Enable:  true,
				Value:   10,
				Uint:    2,
				Float32: 12.3,
			},
		},
		"none of several defaults found warns and continues": {
			Input: input{configPaths: []string{"missing1.toml", "missing2.yaml"}},
			Expected: testStruct{
				Dura:    time.Second,
				Value:   1,
				Uint:    2,
				Float32: 12.3,
			},
		},
		"-c overrides all defaults": {
			Input: input{
				configPaths: []string{"test/test.toml", "missing.toml"},
				flags:       []string{"-c=test/test.yaml"},
			},
			Expected: testStruct{
				Name:    "yaml",
				Time:    trial.Day("2010-08-10"),
				Dura:    10 * time.Second,
				Enable:  true,
				Value:   10,
				Uint:    2,
				Float32: 12.3,
			},
		},
	}
	trial.New(fn, cases).SubTest(t)
}

func TestResolveBinaryPath(t *testing.T) {
	// Fixed fixture: binDir holds a plain file, a subdirectory file, and a symlink
	// that points outside binDir.
	binDir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "config.toml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(binDir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "sub", "nested.toml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret.toml")
	if err := os.WriteFile(secret, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(binDir, "link.toml")); err != nil {
		t.Fatal(err)
	}

	// Resolved paths come back under the symlink-evaluated binDir, so compare
	// against that same base.
	evalBinDir, err := filepath.EvalSymlinks(binDir)
	if err != nil {
		t.Fatal(err)
	}
	fn := func(in string) (string, error) {
		got, err := resolveBinaryPath(binDir, in)
		if err != nil {
			return "", err
		}
		return filepath.Rel(evalBinDir, got)
	}
	cases := trial.Cases[string, string]{
		"plain file": {
			Input:    "config.toml",
			Expected: "config.toml",
		},
		"subdir file": {
			Input:    "sub/nested.toml",
			Expected: filepath.Join("sub", "nested.toml"),
		},
		"missing file returns path under binary dir": {
			Input:    "missing.toml",
			Expected: "missing.toml",
		},
		"rejects absolute path": {
			Input:     "/etc/config.toml",
			ShouldErr: true,
		},
		"rejects path traversal": {
			Input:     "../config.toml",
			ShouldErr: true,
		},
		"rejects symlink escape": {
			Input:     "link.toml",
			ShouldErr: true,
		},
	}
	trial.New(fn, cases).SubTest(t)
}

func TestShow(t *testing.T) {
	fn := func(in interface{}) (string, error) {
		var buf bytes.Buffer
		err := New(in).Show(&buf)
		return buf.String(), err
	}
	// Named so the nested cases can share a shape. Two anonymous struct literals are different types.
	type dbCreds struct {
		User string
		Pass string
	}
	cases := trial.Cases[interface{}, string]{
		"B1: includes unmasked field": {
			Input: &struct {
				Host string
			}{Host: "localhost"},
			Expected: "Host: localhost\n",
		},
		"B1: masks set field": {
			Input: &struct {
				Host     string
				Password string `mask:"true"`
			}{Host: "localhost", Password: "s3cret"},
			Expected: "Host: localhost\nPassword: ****\n",
		},
		"B1: masked zero field prints blank": {
			Input: &struct {
				Host     string
				Password string `mask:"true"`
			}{Host: "localhost"},
			Expected: "Host: localhost\nPassword:\n",
		},
		"B1: nil masked pointer prints nil": {
			Input: &struct {
				Token *string `mask:"true"`
			}{},
			Expected: "Token: <nil>\n",
		},
		"B1: masked struct masks each field": {
			Input: &struct {
				Name string
				DB   dbCreds `mask:"true"`
			}{Name: "app", DB: dbCreds{User: "root", Pass: "s3cret"}},
			Expected: "Name: app\nDB:\n  User: ****\n  Pass: ****\n",
		},
		"B1: masked struct zero field prints blank": {
			Input: &struct {
				DB dbCreds `mask:"true"`
			}{DB: dbCreds{User: "root"}},
			Expected: "DB:\n  User: ****\n  Pass:\n",
		},
	}
	trial.New(fn, cases).SubTest(t)
}

func TestLoadOrDie(t *testing.T) {
	if os.Getenv("GO_CONFIG_LOAD_OR_DIE") == "1" {
		// Not a struct pointer, so Load fails before any config is read.
		New(struct{}{}).LoadOrDie()
		return
	}
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-test.run=^TestLoadOrDie$")
	cmd.Env = append(os.Environ(), "GO_CONFIG_LOAD_OR_DIE=1")
	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("LoadOrDie() error = %v, want exit status 1", err)
	}
}

func TestLoad_twice(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	t.Cleanup(func() {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	})
	region := flag.String("region", "", "library flag")

	type cfg struct{ Name string }
	var first, second cfg

	os.Args = []string{"go-config", "-region=us", "-name=first"}
	if err := New(&first).Disable(OptEnv | OptEnvFile).Load(); err != nil {
		t.Fatalf("first Load() error = %v", err)
	}
	if first.Name != "first" {
		t.Fatalf("first Load() Name = %q, want first", first.Name)
	}
	if *region != "us" {
		t.Fatalf("global flag region = %q, want us", *region)
	}

	os.Args = []string{"go-config", "-name=second"}
	if err := New(&second).Disable(OptEnv | OptEnvFile).Load(); err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if second.Name != "second" {
		t.Fatalf("second Load() Name = %q, want second", second.Name)
	}
}

func TestLoad_reservedFlag(t *testing.T) {
	const want = `flag "show" is already used by go-config (-show). Call Disable(OptShow), or rename the flag with a tag such as flag:"display".`
	reset := func() {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}
	reset()
	t.Cleanup(reset)

	flag.Bool("show", false, "app show")
	os.Args = []string{"go-config"}
	var cfg struct{ Name string }
	err := New(&cfg).Disable(OptEnv | OptEnvFile).Load()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Load() with global -show error = %v, want it to contain %q", err, want)
	}

	reset()
	os.Args = []string{"go-config"}
	var tagged struct {
		Shown bool `flag:"show"`
	}
	err = New(&tagged).Disable(OptEnv | OptEnvFile).Load()
	if err == nil || err.Error() != want {
		t.Fatalf("Load() with flag:\"show\" error = %v, want %q", err, want)
	}

	reset()
	flag.Bool("show", false, "app show")
	os.Args = []string{"go-config", "-show"}
	err = New(&cfg).Disable(OptEnv | OptEnvFile | OptShow).Load()
	if err != nil {
		t.Fatalf("Load() with Disable(OptShow) error = %v", err)
	}
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
