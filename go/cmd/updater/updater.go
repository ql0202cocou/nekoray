package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	maxArchiveFileSize  = 512 * 1024 * 1024
	maxArchiveTotalSize = 1024 * 1024 * 1024
)

func Updater() {
	pre_cleanup := func() {
		if runtime.GOOS == "linux" {
			os.RemoveAll("./usr")
		}
		os.RemoveAll("./nekoray_update")
	}

	// find update package
	var updatePackagePath string
	if len(os.Args) == 2 && Exist(os.Args[1]) {
		updatePackagePath = os.Args[1]
	} else if Exist("./nekoray.zip") {
		updatePackagePath = "./nekoray.zip"
	} else if Exist("./nekoray.tar.gz") {
		updatePackagePath = "./nekoray.tar.gz"
	} else {
		log.Fatalln("no update")
	}
	log.Println("updating from", updatePackagePath)

	// extract update package
	if strings.HasSuffix(updatePackagePath, ".zip") {
		pre_cleanup()
		err := extractZipSafe(updatePackagePath, "./nekoray_update")
		if err != nil {
			log.Fatalln(err.Error())
		}
	} else if strings.HasSuffix(updatePackagePath, ".tar.gz") {
		pre_cleanup()
		err := extractTarGzSafe(updatePackagePath, "./nekoray_update")
		if err != nil {
			log.Fatalln(err.Error())
		}
	} else {
		log.Fatalln("unsupported update package:", updatePackagePath)
	}

	// remove old file
	removeAll("./*.dll")
	removeAll("./*.dmp")

	// update move
	err := Mv("./nekoray_update/nekoray", "./")
	if err != nil {
		MessageBoxPlain("NekoGui Updater", "Update failed. Please close the running instance and run the updater again.\n\n"+err.Error())
		log.Fatalln(err.Error())
	}

	os.RemoveAll("./nekoray_update")
	os.RemoveAll("./nekoray.zip")
	os.RemoveAll("./nekoray.tar.gz")

	// nekoray -> nekobox
	os.Remove("./nekoray.exe")
	os.Remove("./nekoray.png")
	os.Remove("./nekoray_core.exe")
}

func Exist(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func FindExist(paths []string) string {
	for _, path := range paths {
		if Exist(path) {
			return path
		}
	}
	return ""
}

func Mv(src, dst string) error {
	s, err := os.Stat(src)
	if err != nil {
		return err
	}
	if s.IsDir() {
		es, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range es {
			err = Mv(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name()))
			if err != nil {
				return err
			}
		}
	} else {
		err = os.MkdirAll(filepath.Dir(dst), 0755)
		if err != nil {
			return err
		}
		err = os.Rename(src, dst)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeAll(glob string) {
	files, _ := filepath.Glob(glob)
	for _, f := range files {
		os.Remove(f)
	}
}

func safeJoin(root, archivePath string) (string, error) {
	name := strings.ReplaceAll(archivePath, "\\", "/")
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return "", fmt.Errorf("archive path %q escapes destination", archivePath)
		}
	}
	clean := path.Clean(name)
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid archive path %q", archivePath)
	}
	if path.IsAbs(clean) || filepath.IsAbs(filepath.FromSlash(clean)) {
		return "", fmt.Errorf("absolute archive path %q is not allowed", archivePath)
	}
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", fmt.Errorf("archive path %q escapes destination", archivePath)
	}
	if len(clean) >= 2 && clean[1] == ':' {
		return "", fmt.Errorf("archive path %q contains a drive prefix", archivePath)
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := filepath.Join(rootAbs, filepath.FromSlash(clean))
	rel, err := filepath.Rel(rootAbs, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("archive path %q escapes destination", archivePath)
	}
	return target, nil
}

func extractZipSafe(src, dst string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	var total int64
	for _, zf := range zr.File {
		target, err := safeJoin(dst, zf.Name)
		if err != nil {
			return err
		}

		mode := zf.FileInfo().Mode()
		if mode&os.ModeSymlink != 0 {
			return fmt.Errorf("archive symlink %q is not allowed", zf.Name)
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if !mode.IsRegular() {
			return fmt.Errorf("archive entry %q has unsupported mode %s", zf.Name, mode.String())
		}
		if zf.UncompressedSize64 > maxArchiveFileSize {
			return fmt.Errorf("archive entry %q exceeds max file size", zf.Name)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		rc, err := zf.Open()
		if err != nil {
			return err
		}
		n, err := writeArchiveFile(target, rc, archiveFilePerm(mode), maxArchiveFileSize)
		closeErr := rc.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		total += n
		if total > maxArchiveTotalSize {
			os.Remove(target)
			return errors.New("archive exceeds max total extracted size")
		}
	}
	return nil
}

func extractTarGzSafe(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	var total int64
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		target, err := safeJoin(dst, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if hdr.Size < 0 || hdr.Size > maxArchiveFileSize {
				return fmt.Errorf("archive entry %q exceeds max file size", hdr.Name)
			}
			if total+hdr.Size > maxArchiveTotalSize {
				return errors.New("archive exceeds max total extracted size")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			n, err := writeArchiveFile(target, tr, archiveFilePerm(os.FileMode(hdr.Mode)), hdr.Size)
			if err != nil {
				return err
			}
			total += n
		default:
			return fmt.Errorf("archive entry %q has unsupported type %d", hdr.Name, hdr.Typeflag)
		}
	}
	return nil
}

func archiveFilePerm(mode os.FileMode) os.FileMode {
	if mode&0111 != 0 {
		return 0755
	}
	return 0644
}

func writeArchiveFile(target string, r io.Reader, perm os.FileMode, limit int64) (int64, error) {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := copyLimited(f, r, limit)
	if err != nil {
		os.Remove(target)
		return n, err
	}
	return n, nil
}

func copyLimited(dst io.Writer, src io.Reader, limit int64) (int64, error) {
	lr := &io.LimitedReader{R: src, N: limit + 1}
	n, err := io.Copy(dst, lr)
	if err != nil {
		return n, err
	}
	if n > limit {
		return n, fmt.Errorf("entry exceeds limit of %d bytes", limit)
	}
	return n, nil
}
