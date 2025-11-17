package filesystem

import "os"

func WriteFile(path string, content []byte, perm os.FileMode) error {
	if perm == 0 {
		perm = privateFilePermission
	}

	if err := os.WriteFile(path, content, perm); err != nil {
		return err
	}

	return nil
}

func ReadFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return content, nil
}
