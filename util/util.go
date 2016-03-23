package util

import (
	"fmt"
	"os"
)

func FindOrCreateDisk(path string, size int64) error {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("Cannot create new disk file %v, due to %v", path, err.Error())
		}
		if err := file.Truncate(size); err != nil {
			return fmt.Errorf("Cannot resize new disk file %v, due to %v", path, err.Error())
		}
		file.Close()

		stat, err = os.Stat(path)
		if err != nil {
			return err
		}

	}
	if stat.IsDir() {
		return fmt.Errorf("Cannot find disk file %v, it's a directory", path)
	}
	/*
		if stat.Size() != size {
			return fmt.Errorf("Disk file %v size %v is not the same as %v",
				path, stat.Size(), size)
		}
	*/
	return nil
}
