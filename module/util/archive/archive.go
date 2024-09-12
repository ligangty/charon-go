package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"org.commonjava/charon/module/util/files"
	"org.commonjava/charon/module/util/httpc"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func ExtractZipAll(zipFilePath, targetDir string) error {
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		logger.Error(fmt.Sprintf("impossible to open zip reader: %s", err))
		return err
	}
	defer r.Close()
	for k, f := range r.File {
		logger.Debug(fmt.Sprintf("Unzipping %s:\n", f.Name))
		rc, err := f.Open()
		if err != nil {
			logger.Error(fmt.Sprintf("impossible to open file n°%d in archine: %s", k, err))
		}
		defer rc.Close()
		newFilePath := path.Join(targetDir, f.Name)

		if f.FileInfo().IsDir() {
			err = os.MkdirAll(newFilePath, 0755)
			if err != nil {
				logger.Error(fmt.Sprintf("impossible to MkdirAll: %s", err))
			}
		} else {
			uncompressedFile, err := os.Create(newFilePath)
			if err != nil {
				logger.Error(fmt.Sprintf("impossible to create uncompressed: %s", err))
			}
			_, err = io.Copy(uncompressedFile, rc)
			if err != nil {
				logger.Error(fmt.Sprintf("impossible to copy file n°%d: %s", k, err))
			}
		}
	}
	return nil
}

// Detects, if the archive needs to have npm workflow.
//
// :parameter repo repository directory
//
// :return NpmArchiveType value
func DetectNPMArchive(repo string) NpmArchiveType {
	if !files.FileOrDirExists(repo) {
		logger.Error(fmt.Sprintf("Repository %s does not exist!", repo))
		os.Exit(1)
	}
	if !files.IsFile(repo) {
		repoPath := path.Join(repo, "package.json")
		if files.IsFile(repoPath) {
			return DIRECTORY
		}
	}
	aType, _ := checkArchiveType(repo)
	if aType == "zip" {
		fInfo, _ := getZipFileInfo(repo, "/package.json")
		if fInfo != nil {
			return ZIP_FILE
		}
	} else if aType == "tar" || aType == "tgz" {
		fInfo, _ := getTarFileInfo(repo, "package/package.json")
		if fInfo != nil {
			return TAR_FILE
		}
	}
	return NOT_NPM
}

func DownloadArchive(url, baseDir string) string {
	dir := baseDir
	var err error
	if strings.TrimSpace(dir) == "" || !files.FileOrDirExists(dir) || files.IsFile(dir) {
		dir, err = os.MkdirTemp("", "charon-*")
		if err != nil {
			logger.Error(fmt.Sprintf(
				"Can not create temporary directory for archive download, error: %s ", err))
			return ""
		}
		logger.Info(fmt.Sprintf(
			"No base dir specified for holding archive. Will use a temp dir %s to hold archive", dir))
	}
	urlParts := strings.Split(url, "/")
	localFile := path.Join(dir, urlParts[len(urlParts)-1])
	downloaded, err := httpc.DownloadFile(url, localFile, nil)
	if err != nil {
		logger.Error(
			fmt.Sprintf("Cannot download %s to %s", url, localFile))
		return ""
	}
	return downloaded
}

// Extract npm tarball will relocate the tgz file and metadata files.
//
// * Locate tar path ( e.g.: jquery/-/jquery-7.6.1.tgz or @types/jquery/-/jquery-2.2.3.tgz).
//
// * Locate version metadata path (e.g.: jquery/7.6.1 or @types/jquery/2.2.3).
//
// Result returns the version meta file path and is for following package meta generating.
// func ExtractNPMTarball(repo, targetDir, packageRoot, registry string, isForUpload bool) {
// 	pkgRoot := packageRoot
// 	if strings.TrimSpace(pkgRoot) == "" {
// 		pkgRoot = "package"
// 	}
// 	reg := registry
// 	if strings.TrimSpace(reg) == "" {
// 		reg = DEFAULT_REGISTRY
// 	}

// 	valid_paths := []string{}
// 	package_name_path := ""
// 	tgz = tarfile.open(path)
// 	pkg_file = None
// 	root_pkg_file_exists := true

// 	rootPkgPath := path.Join(pkgRoot, "package.json")
// 	logger.Debug(rootPkgPath)
// }

func checkArchiveType(repo string) (string, error) {

	file, err := os.Open(repo)

	if err != nil {
		logger.Error(fmt.Sprintf("Can not open file %s, error: %s", repo, err))
		return "", err
	}

	defer file.Close()

	buff := make([]byte, 512)

	// why 512 bytes ? see http://golang.org/module/net/http/#DetectContentType

	_, err = file.Read(buff)

	if err != nil {
		logger.Error(fmt.Sprintf("Can read file %s, error: %s", repo, err))
		return "", err
	}

	filetype := http.DetectContentType(buff)

	logger.Debug(fmt.Sprintf("File type is %s", filetype))
	switch filetype {
	case "application/zip":
		return "zip", nil
	case "application/tar", "application/x-tar":
		return "tar", nil
	case "application/x-gzip":
		return "tgz", nil
	default:
		return "", nil
	}
}

func getZipFileInfo(zipRepo, zipEntry string) (os.FileInfo, error) {
	r, err := zip.OpenReader(zipRepo)
	if err != nil {
		logger.Error(fmt.Sprintf("can not open zip %s: %s", zipRepo, err))
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == zipEntry {
			return f.FileInfo(), nil
		}
	}
	return nil, nil
}

func getTarFileInfo(tarRepo, tarEntry string) (os.FileInfo, error) {
	aType, _ := checkArchiveType(tarRepo)
	if aType != "tar" && aType != "tgz" {
		return nil, nil
	}
	f, err := os.Open(tarRepo)
	if err != nil {
		logger.Error(fmt.Sprintf("can not open archive %s: %s", tarRepo, err))
		return nil, err
	}
	defer f.Close()
	var tr *tar.Reader
	if aType == "tgz" {
		gtr, err := gzip.NewReader(f)
		if err != nil {
			logger.Error(fmt.Sprintf("can not open archive %s: %s", tarRepo, err))
			return nil, err
		}
		tr = tar.NewReader(gtr)
	} else {
		tr = tar.NewReader(f)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			logger.Error(fmt.Sprintf("Can not read entry tar %s, error: %s", tarRepo, err))
		}
		fmt.Printf("Contents of %s:\n", hdr.Name)
		if hdr.Name == tarEntry {
			return hdr.FileInfo(), nil
		}
	}

	return nil, nil
}

type NpmArchiveType int

const (
	NOT_NPM   NpmArchiveType = 0
	DIRECTORY NpmArchiveType = 1
	ZIP_FILE  NpmArchiveType = 2
	TAR_FILE  NpmArchiveType = 3
)
