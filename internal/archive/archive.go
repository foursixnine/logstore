package archive

import (
	"archive/tar"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
)

type Archive struct {
	path string
	name string
}

type FileSystem struct {
	fs.FS
}

func NewArchive(path string, name string) *Archive {
	return &Archive{
		path: path,
		name: name,
	}
}

func (ar *Archive) Generate() error {

	root, err := os.OpenRoot(ar.path)
	if err != nil {
		return err
	}

	fileList, err := listFiles(root)
	if err != nil {
		return err
	}

	archive, err := ar.create()
	if err != nil {
		return err
	}
	defer archive.Close()

	tw := tar.NewWriter(archive)
	defer tw.Close()

	if err := appendToArchive(fileList, root, tw); err != nil {
		return err
	}

	archiveInfo, err := archive.Stat()
	if err != nil {
		return err
	}
	message := fmt.Sprintf("Wrote %d bytes to %s", archiveInfo.Size(), ar.Name())
	slog.Debug(message)

	return nil

}

func (ar *Archive) Destroy() error {
	if err := os.Remove(ar.Name()); err != nil {
		return err
	}

	return nil
}

func appendToArchive(fileList []string, root *os.Root, tw *tar.Writer) error {
	slog.Debug("Appending files to archive", "files", fileList)
	for _, file := range fileList {
		fileInfo, err := fs.Stat(root.FS(), file)
		if err != nil {
			return err
		}

		tarHeader, err := tar.FileInfoHeader(fileInfo, "")
		if err != nil {
			return err
		}

		if err := tw.WriteHeader(tarHeader); err != nil {
			return err
		}

		fileBuffer, err := fs.ReadFile(root.FS(), file)
		if err != nil {
			return err
		}

		written, err := tw.Write(fileBuffer)
		if err != nil {
			return err
		}

		slog.Debug("Added file to archive", "name", file, "size", written)

	}
	return nil
}

func (ar Archive) Name() string {
	return path.Join(ar.path, ar.name)
}

func (ar Archive) create() (*os.File, error) {
	file, err := os.Create(ar.Name())
	if err != nil {
		return nil, err
	}

	return file, nil
}

func listFiles(root *os.Root) ([]string, error) {
	var fileNames []string

	err := fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		fileNames = append(fileNames, path)

		return nil
	})

	if err != nil {
		return fileNames, err
	}

	return fileNames, nil
}
