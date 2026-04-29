package archive

import (
	"archive/tar"
	"io/fs"
	"log"
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
	log.Printf("Wrote %d bytes to %s", archiveInfo.Size(), ar.Name())

	if err := tw.Close(); err != nil {
		log.Fatal(err)
	}

	return nil

}

func (ar *Archive) Destroy() error {
	if err := os.Remove(ar.Name()); err != nil {
		return err
	}

	return nil
}

func appendToArchive(fileList []string, root *os.Root, tw *tar.Writer) error {
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
			log.Fatal(err)
		}

		log.Printf("Header of %s is %d", tarHeader.Name, tarHeader.Size)

		fileBuffer, err := fs.ReadFile(root.FS(), file)
		if err != nil {
			return err
		}

		log.Printf("%s has a size of %d", file, len(fileBuffer))

		written, err := tw.Write(fileBuffer)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Added '%s' (%d)", file, written)

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
			log.Printf("Error walking directory: %v", err)
			return err
		}

		if d.IsDir() {
			log.Print("Skipping directory")
			return nil
		}

		log.Printf("Walking on %s", path)

		fileNames = append(fileNames, path)

		return nil
	})

	if err != nil {
		log.Println("Unexpected error")
		return fileNames, err
	}

	return fileNames, nil
}
