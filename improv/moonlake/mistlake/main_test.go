package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandProcess(t *testing.T) {
	if os.Getenv("MISTLAKE_TEST_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"mistlake"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	os.Exit(99)
}

func command(t *testing.T, args []string, vars map[string]string) ([]byte, int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestCommandProcess$", "--"}, args...)...)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "MIST_COLOUR", "MIST_THICK", "MOONREF", "WATERMASK", "TREEMASK", "MISTLAKE_TEST_PROCESS":
		default:
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "MISTLAKE_TEST_PROCESS=1")
	for key, value := range vars {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return out, 0
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return out, exit.ExitCode()
	}
	t.Fatal(err)
	return nil, -1
}

func fixture(t *testing.T, dir, name string, scale float64, mode int) string {
	t.Helper()
	const width, height = 7, 80
	var data bytes.Buffer
	fmt.Fprintf(&data, "PF\n%d %d\n%g\n", width, height, scale)
	pixels := make([]float32, width*height*3)
	for i := range pixels {
		switch mode {
		case 0:
			pixels[i] = float32(i%23+1) / 17
		case 1:
			pixels[i] = float32(i%13+2) * 3
		case 2:
			if i/3%4 == 0 {
				pixels[i] = float32(i%7+1) / 9
			}
		case 3:
			if i/3%11 == 0 {
				pixels[i] = float32(i%5+1) / 13
			}
		}
	}
	var order binary.ByteOrder = binary.LittleEndian
	if scale > 0 {
		order = binary.BigEndian
	}
	if err := binary.Write(&data, order, pixels); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCommandBytes(t *testing.T) {
	for _, tc := range []struct {
		name                string
		scale               float64
		mode                int
		seed, seconds, want string
	}{
		{"little-default", -1.25, 0, "23", "0", "9ee432a0cbd5cb9544695a0c387d0c606f01eed25f86b0034683565f2a2c6f39"},
		{"big-default", 2.5, 0, "23", "0", "271cc8daf0c9a5b79a6582bbf2da9a0c022cad5f71c75184c89d2369ebeb391d"},
		{"colour-layers-time", -1.25, 1, "7920", "13.25", "552ad7b913f77c142eb670fe29c3e11d652132e98ebf4302b800438b5400bafd"},
		{"moon-reference", -1.25, 2, "23", "0", "3f2c90492355793610f6bebc276073ec9904e5f4a07af1aa076b1890291c8deb"},
		{"water", -1.25, 3, "23", "0", "31169fc230ca3281fdd3d289e4035ba722c280cea22408220edfabf6130cf3b5"},
		{"trees", -1.25, 4, "23", "0", "faa32c35b609c857996adfedb0cb948ff04134f4e9af74b2d0dca3fd1a428259"},
		{"all-big", 2.5, 5, "7920", "-7.5", "132662bacc98e1610ff92d97c83db5240c6172d31b0b3d4893bee171e272e75d"},
		{"fractional-seed", -1.25, 0, "23.75", "0", "9ee432a0cbd5cb9544695a0c387d0c606f01eed25f86b0034683565f2a2c6f39"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			raw := fixture(t, dir, "raw.pfm", tc.scale, 0)
			out := filepath.Join(dir, "out.pfm")
			vars := map[string]string{}
			if tc.mode == 1 || tc.mode == 5 {
				vars["MIST_COLOUR"] = "0.31, 1.13, 0.87"
				vars["MIST_THICK"] = "3.5"
			}
			if tc.mode == 2 || tc.mode == 5 {
				vars["MOONREF"] = fixture(t, dir, "ref.pfm", 1, 1)
			}
			if tc.mode == 3 || tc.mode == 5 {
				vars["WATERMASK"] = fixture(t, dir, "water.pfm", -1, 2)
			}
			if tc.mode == 4 || tc.mode == 5 {
				vars["TREEMASK"] = fixture(t, dir, "trees.pfm", 1, 3)
			}
			text, status := command(t, []string{raw, out, "6", "-6", "0", "0.43", tc.seed, tc.seconds}, vars)
			if status != 0 || len(text) != 0 {
				t.Fatalf("status %d, output %q", status, text)
			}
			data, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			got := fmt.Sprintf("%x", sha256.Sum256(data))
			if got != tc.want {
				t.Fatalf("output hash %s, want %s", got, tc.want)
			}
		})
	}
}

func TestCommandErrors(t *testing.T) {
	dir := t.TempDir()
	raw := fixture(t, dir, "raw.pfm", -1, 0)
	missing := filepath.Join(dir, "missing.pfm")
	bad := filepath.Join(dir, "bad.pfm")
	if err := os.WriteFile(bad, []byte("Pf\n1 1\n-1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	short := filepath.Join(dir, "short.pfm")
	if err := os.WriteFile(short, []byte("PF\n1 1\n-1\nx"), 0600); err != nil {
		t.Fatal(err)
	}
	base := []string{raw, filepath.Join(dir, "out.pfm"), "6", "-6", "0", "0.43", "23", "0"}
	for _, tc := range []struct {
		name   string
		index  int
		value  string
		vars   map[string]string
		status int
		want   string
	}{
		{"usage", -1, "", nil, 2, "mistlake in.pfm out.pfm top bottom shore strength seed seconds\n"},
		{"number", 2, "bad", nil, 1, "mistlake: strconv.ParseFloat: parsing \"bad\": invalid syntax\n"},
		{"raw", 0, missing, nil, 1, "mistlake: open " + missing + ": no such file or directory\n"},
		{"kind", 0, bad, nil, 1, "mistlake: " + bad + " is Pf, not a colour pfm\n"},
		{"truncated", 0, short, nil, 1, "mistlake: unexpected EOF\n"},
		{"reference", -2, "", map[string]string{"MOONREF": missing}, 1, "mistlake: open " + missing + ": no such file or directory\n"},
		{"water", -2, "", map[string]string{"WATERMASK": missing}, 1, "mistlake: open " + missing + ": no such file or directory\n"},
		{"trees", -2, "", map[string]string{"TREEMASK": missing}, 1, "mistlake: open " + missing + ": no such file or directory\n"},
		{"output", 1, dir, nil, 1, "mistlake: open " + dir + ": is a directory\n"},
		{"colour-count", -1, "", map[string]string{"MIST_COLOUR": "1,2"}, 2, "MIST_COLOUR=\"1,2\" is not r,g,b\n"},
		{"colour-number", -1, "", map[string]string{"MIST_COLOUR": "1,no,3"}, 2, "MIST_COLOUR=\"1,no,3\" is not r,g,b\n"},
		{"layers-number", -1, "", map[string]string{"MIST_THICK": "no"}, 2, "MIST_THICK=\"no\" is not a number of at least one\n"},
		{"layers-small", -1, "", map[string]string{"MIST_THICK": "0.5"}, 2, "MIST_THICK=\"0.5\" is not a number of at least one\n"},
		{"env-order", -1, "", map[string]string{"MIST_THICK": "no", "MIST_COLOUR": "bad"}, 2, "MIST_COLOUR=\"bad\" is not r,g,b\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string(nil), base...)
			if tc.index == -1 {
				args = nil
			} else if tc.index >= 0 {
				args[tc.index] = tc.value
			}
			got, status := command(t, args, tc.vars)
			if status != tc.status || string(got) != tc.want {
				t.Fatalf("status %d, output %q; want status %d, output %q", status, got, tc.status, tc.want)
			}
		})
	}
}
