package cache

import (
	"encoding/json"
	"github.com/ArrowSK/ha-energy-security/energy_security/internal/model"
	"os"
	"path/filepath"
)

type Store struct{ Path string }

func (s Store) Load() (model.Snapshot, error) {
	var v model.Snapshot
	b, err := os.ReadFile(s.Path)
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(b, &v)
	return v, err
}
func (s Store) Save(v model.Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
