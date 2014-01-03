package opstocat

import (
	"io/ioutil"
	"path"
	"testing"
)

func TestCurrentShaFromFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "opstocat-test-")
	if err != nil {
		t.Fatal("Expected to be able to create a temporal directory")
	}
	sha := "4628f8f626af95168d7139dde4c6e503bd0acf53"
	ioutil.WriteFile(path.Join(dir, "SHA1"), []byte(sha), 0755)

	newSha := currentSha(dir)
	if newSha != sha {
		t.Errorf("Expected current sha to be %s, but it was %s", sha, newSha)
	}
}
