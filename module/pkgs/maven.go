package pkgs

import (
	"bytes"
	"crypto"
	"encoding/xml"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"org.commonjava/charon/module/config"
	"org.commonjava/charon/module/storage"
	"org.commonjava/charon/module/util"
	"org.commonjava/charon/module/util/archive"
	"org.commonjava/charon/module/util/files"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

// This MavenMetadata will represent a maven-metadata.xml data content
// which will be used in jinja2 or other places
type MavenMetadata struct {
	GroupId        string
	ArtifactId     string
	LastUpdateTime string
	versions       []string
	latestVersion  string
	releaseVersion string
}

func (m *MavenMetadata) LatestVersion() string {
	if !util.IsBlankString(m.latestVersion) {
		return m.latestVersion
	}
	versions := m.Versions()
	m.latestVersion = versions[len(versions)-1]
	return m.latestVersion
}
func (m *MavenMetadata) ReleaseVersion() string {
	if !util.IsBlankString(m.releaseVersion) {
		return m.releaseVersion
	}
	versions := m.Versions()
	m.releaseVersion = versions[len(versions)-1]
	return m.releaseVersion
}
func (m *MavenMetadata) Versions() []string {
	vers := m.versions
	slices.SortFunc(vers, versionCompare)
	m.versions = vers
	return m.versions
}
func (m *MavenMetadata) String() string {
	return fmt.Sprintf("%s:%s:\n%s\n\n", m.GroupId, m.ArtifactId, m.Versions())
}
func (m *MavenMetadata) GenerateMetaFileContent() (string, error) {
	t := template.Must(template.New("settings").Parse(MAVEN_METADATA_TEMPLATE))
	var buf bytes.Buffer
	err := t.Execute(&buf, m)
	if err != nil {
		logger.Error(fmt.Sprintf("executing template: %s", err))
		return "", err
	}
	return buf.String(), nil
}

// This ArchetypeRef will represent an entry in archetype-catalog.xml content
// which will be used in jinja2 or other places
type ArchetypeRef struct {
	GroupId     string `xml:"groupId"`
	ArtifactId  string `xml:"artifactId"`
	Version     string `xml:"version"`
	Repository  string `xml:"repository"`
	Description string `xml:"description"`
}

func (m ArchetypeRef) String() string {
	return fmt.Sprintf("%s:%s\n%s\n%s\n\n",
		m.GroupId, m.ArtifactId, m.Version, m.Description)
}

// This MavenArchetypeCatalog represents an archetype-catalog.xml which will be
// used in jinja2 to regenerate the file with merged contents
type MavenArchetypeCatalog struct {
	Archetypes []ArchetypeRef `xml:"archetypes>archetype"`
}

func NewMavenArchetypeCatalog(archetypes []ArchetypeRef) MavenArchetypeCatalog {
	archs := make([]ArchetypeRef, len(archetypes))
	copy(archs, archetypes)
	slices.SortFunc(archs, archetypeRefCompare)
	return MavenArchetypeCatalog{Archetypes: archs}
}

func (m *MavenArchetypeCatalog) GenerateMetaFileContent() (string, error) {
	bytes, err := xml.MarshalIndent(m, "", "  ")
	if err != nil {
		logger.Error(fmt.Sprintf("executing template: %s", err))
		return "", err
	}
	return string(bytes), nil
	// t := template.Must(template.New("archetype").Parse(ARCHETYPE_CATALOG_TEMPLATE))
	// var buf bytes.Buffer
	// err := t.Execute(&buf, m)
	// if err != nil {
	// 	logger.Error(fmt.Sprintf("executing template: %s", err))
	// 	return "", err
	// }
	// return buf.String(), nil
}

func (m *MavenArchetypeCatalog) String() string {
	return fmt.Sprintf("(Archetype Catalog with %v entries).\n\n", len(m.Archetypes))
}

// Handle the maven product release tarball uploading process.
//   - repo is the location of the tarball in filesystem
//   - prod_key is used to identify which product this repo
//     tar belongs to
//   - ignore_patterns is used to filter out paths which don't
//     need to upload in the tarball
//   - root is a prefix in the tarball to identify which path is
//     the beginning of the maven GAV path
//   - targets contains the target name with its bucket name and prefix
//     for the bucket, which will be used to store artifacts with the
//     prefix. See target definition in Charon configuration for details
//   - dir_ is base dir for extracting the tarball, will use system
//     tmp dir if None.
//
// Returns the directory used for archive processing and if the uploading is successful
func HandleMavenUploading(
	repo,
	prodKey string,
	ignorePatterns []string,
	root string,
	targets []config.Target,
	awsProfile,
	dir_ string,
	doIndex,
	genSign bool,
	cfEnable bool,
	key string,
	dryRun bool,
	manifestBucketName,
	configFilePath string,
) (string, bool) {
	realRoot := root
	if util.IsBlankString(realRoot) {
		realRoot = "maven-repository"
	}
	// step 1. extract tarball
	tmpRoot := extractTarball(repo, prodKey, dir_)

	// step 2. scan for paths and filter out the ignored paths,
	// and also collect poms for later metadata generation
	scannedPaths := scanPaths(ignorePatterns, tmpRoot, root)
	validMvnPaths, topLevel := scannedPaths.mvnPaths, scannedPaths.topLevel

	// This prefix is a subdir under top-level directory in tarball
	// or root before real GAV dir structure
	if !files.IsDir(topLevel) {
		panic(fmt.Errorf("error: the extracted top-level path %s does not exist",
			topLevel))
	}

	// step 3. do validation for the files, like product version checking
	logger.Info("Validating paths with rules.")
	errMsgs, passed := validateMaven(validMvnPaths)
	if !passed {
		handleError(errMsgs)
		//Question: should we exit here?
	}

	// step 4. Do uploading
	s3Client, err := storage.NewS3Client(
		awsProfile, storage.DEFAULT_CONCURRENT_LIMIT, dryRun)
	if err != nil {
		panic(err)
	}
	fixedTargets := make([]config.Target, len(targets))
	buckets := make([]string, len(targets))
	for i, t := range targets {
		fixedTargets[i] = config.Target{
			Bucket:   t.Bucket,
			Prefix:   strings.TrimPrefix(t.Prefix, "/"),
			Registry: t.Registry,
			Domain:   t.Domain,
		}
		buckets[i] = t.Bucket
	}
	logger.Info(fmt.Sprintf("Start uploading files to s3 buckets: %s", buckets))
	failedFiles := s3Client.UploadFiles(
		validMvnPaths, fixedTargets, prodKey, topLevel)
	logger.Info("Files uploading done\n")
	succeeded := true
	generatedSigns := []string{}
	for _, t := range fixedTargets {
		// prepare cf invalidate files
		cfInvalidatePaths := []string{}
		// step 5. Do manifest uploading
		if util.IsBlankString(manifestBucketName) {
			logger.Warn("Warning: No manifest bucket is provided, will ignore the process of manifest uploading\n")
		} else {
			logger.Info("Start uploading manifest to s3 bucket " + manifestBucketName)
			manifestFolder := t.Bucket
			manifestName, manifestFullPath := files.WriteManifest(validMvnPaths, topLevel, prodKey)
			s3Client.UploadManifest(manifestName, manifestFullPath, manifestFolder, manifestBucketName)
			logger.Info("Manifest uploading is done\n")
		}

		// step 6. Use uploaded poms to scan s3 for metadata refreshment
		bucketName := t.Bucket
		prefix := t.Prefix
		validPoms := scannedPaths.poms
		logger.Info("Start generating maven-metadata.xml files for bucket " + bucketName)
		metaFiles := generateMetadatas(*s3Client, validPoms, bucketName, prefix, root)
		logger.Info("maven-metadata.xml files generation done\n")
		failedMetas := metaFiles[META_FILE_FAILED]

		// step 7. Upload all maven-metadata.xml
		if v, ok := metaFiles[META_FILE_GEN_KEY]; ok {
			logger.Info("Start updating maven-metadata.xml to s3 bucket " + bucketName)
			_failedMetas := s3Client.UploadMetadatas(v, t, "", topLevel)
			failedMetas = append(failedMetas, _failedMetas...)
			logger.Info(
				fmt.Sprintf("maven-metadata.xml updating done in bucket %s\n", bucketName))
			// Add maven-metadata.xml to CF invalidate paths
			if cfEnable {
				cfInvalidatePaths = append(cfInvalidatePaths, metaFiles[META_FILE_GEN_KEY]...)
			}
		}

		// step 8. Determine refreshment of archetype-catalog.xml
		if files.FileOrDirExists(path.Join(topLevel, MAVEN_ARCH_FILE)) {
			logger.Info("Start generating archetype-catalog.xml for bucket " + bucketName)
			uploadArchetypeFile := generateUploadArchetypeCatalog(s3Client, bucketName, topLevel, prefix)
			logger.Info(
				fmt.Sprintf("archetype-catalog.xml files generation done in bucket %s\n", bucketName))
			if uploadArchetypeFile {
				archetypeFiles := []string{path.Join(topLevel, MAVEN_ARCH_FILE)}
				archetypeFiles = append(archetypeFiles, hashDecorateMetadata(topLevel, MAVEN_ARCH_FILE)...)
				logger.Info("Start updating archetype-catalog.xml to s3 bucket %s" + bucketName)
				_failedMetas := s3Client.UploadMetadatas(archetypeFiles, t, "", topLevel)
				failedMetas = append(failedMetas, _failedMetas...)
				logger.Info(fmt.Sprintf("archetype-catalog.xml updating done in bucket %s\n", bucketName))
				// Add archtype-catalog to invalidate paths
				if cfEnable {
					cfInvalidatePaths = append(cfInvalidatePaths, archetypeFiles...)
				}
			}
		}

		// step 10. Generate signature file if contain_signature is set to True
		if genSign {
			conf, err := config.GetConfig(configFilePath)
			if err != nil {
				panic(err)
			}
			suffixList := getSuffix(PACKAGE_TYPE_MAVEN, *conf)
			command := conf.SignatureCommand
			artifacts := []string{}
			for _, p := range validMvnPaths {
				suffixed := false
				for _, s := range suffixList {
					if strings.HasSuffix(p, s) {
						suffixed = true
						break
					}
				}
				if !suffixed {
					artifacts = append(artifacts, p)
				}
			}
			logger.Info(
				fmt.Sprintf("Start generating signature for s3 bucket %s\n", bucketName))
			_failedMetas, _generatedSigns := generateSign(
				*s3Client, artifacts, util.PACKAGE_TYPE_MAVEN,
				topLevel, prefix, bucketName, key, command)
			failedMetas = append(failedMetas, _failedMetas...)
			generatedSigns = append(generatedSigns, _generatedSigns...)
			logger.Info("Singature generation done.\n")
			logger.Info(
				fmt.Sprintf("Start upload singature files to s3 bucket %s\n", bucketName))
			_failedMetas = s3Client.UploadSignatures(
				generatedSigns, t, "", topLevel)
			failedMetas = append(failedMetas, _failedMetas...)
			logger.Info("Signature uploading done.\n")
		}

		//  this step generates index.html for each dir and add them to file list
		//  index is similar to metadata, it will be overwritten everytime
		validDirs := scannedPaths.dirs
		if doIndex {
			logger.Info("Start generating index files to s3 bucket " + bucketName)
			createdIndex := generateIndexes(*s3Client, validDirs,
				PACKAGE_TYPE_MAVEN, topLevel, bucketName, prefix)
			logger.Info("Index files generation done.\n")
			logger.Info("Start updating index files to s3 bucket " + bucketName)
			_failed_metas := s3Client.UploadMetadatas(createdIndex, t, prodKey, topLevel)
			failedMetas = append(failedMetas, _failed_metas...)
			logger.Info("Index files updating done\n")
			// We will not invalidate the index files per cost consideration
			// if cfEnable {
			// 	cfInvalidatePaths = append(cfInvalidatePaths, createdIndex...)
			// }
		} else {
			logger.Info("Bypass indexing")
		}

		// step 11. Finally do the CF invalidating for metadata files
		if cfEnable && len(cfInvalidatePaths) > 0 {
			cfClient, err := storage.NewCFClient(awsProfile)
			if err != nil {
				logger.Error(
					fmt.Sprintf("Cannot do Cloudfront cache invalidating due to error: %s", err))
			} else {
				cfInvalidatePaths = wildcardMetadataPaths(cfInvalidatePaths)
				invalidateCFPaths(cfClient, t, cfInvalidatePaths, root, storage.INVALIDATION_BATCH_DEFAULT)
			}
		}

		uploadPostProcess(failedFiles, failedMetas, prodKey, bucketName)
		succeeded = succeeded && len(failedFiles) <= 0 && len(failedMetas) <= 0
	}

	return tmpRoot, succeeded
}

func isInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// Scan a file path and finds all pom files absolute paths
func scanForPoms(fullPath string) []string {
	allPomPaths := []string{}
	filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && filepath.Ext(path) == ".pom" {
			allPomPaths = append(allPomPaths, path)
		}
		return nil
	})
	return allPomPaths
}

// Parse maven groupId and artifactId from a standard path in a local maven repo.
//
// e.g: org/apache/maven/plugin/maven-plugin-plugin -> (org.apache.maven.plugin,
// maven-plugin-plugin)
//
// root is like a prefix of the path which is not part of the maven GAV
func parseGA(fullGAPath, root string) [2]string {
	gaPath := trimRoot(fullGAPath, root)

	items := strings.Split(gaPath, "/")
	artifact := items[len(items)-1]
	group := strings.Join(items[:len(items)-1], ".")

	return [2]string{group, artifact}
}

// Parse maven groupId, artifactId and version from a standard path in a local maven repo.
//
// e.g: org/apache/maven/plugin/maven-plugin-plugin/1.0.0/maven-plugin-plugin-1.0.0.pom
// -> (org.apache.maven.plugin, maven-plugin-plugin, 1.0.0)
//
// root is like a prefix of the path which is not part of the maven GAV
func parseGAV(fullArtifactPath, root string) [3]string {
	verPath := trimRoot(fullArtifactPath, root)

	items := strings.Split(verPath, "/")
	version := items[len(items)-2]
	artifact := items[len(items)-3]
	group := strings.Join(items[:len(items)-3], ".")

	return [3]string{group, artifact, version}
}

func trimRoot(fullPath, root string) string {
	fixedRoot := fixRoot(root)
	if !strings.HasSuffix(fixedRoot, "/") {
		fixedRoot += "/"
	}

	verPath := strings.TrimPrefix(fullPath, fixedRoot)
	verPath = strings.TrimSuffix(verPath, "/")
	return verPath
}

// Give a list of paths with pom files and parse the maven groupId, artifactId and version
// from them. The result will be a dict like {groupId: {artifactId: [versions list]}}.
// Root is like a prefix of the path which is not part of the maven GAV
func parseGAVs(pomPaths []string, root string) map[string]map[string][]string {
	gavs := make(map[string]map[string][]string)
	for _, pom := range pomPaths {
		gav := parseGAV(pom, root)
		g := gav[0]
		a := gav[1]
		v := gav[2]
		avs := make(map[string][]string)
		if item, ok := gavs[g]; ok {
			avs = item
		}
		vers := []string{}
		if item, ok := avs[a]; ok {
			vers = item
		}
		vers = append(vers, v)
		avs[a] = vers
		gavs[g] = avs
	}
	return gavs
}

func genMetaFile(groupId, artifactId string,
	versions []string, root string, digest bool) ([]string, error) {
	fixedRoot := fixRoot(root)
	meta := &MavenMetadata{
		GroupId:    groupId,
		ArtifactId: artifactId,
		versions:   versions,
	}
	content, err := meta.GenerateMetaFileContent()
	if err != nil {
		return []string{}, err
	}

	gPath := strings.Join(strings.Split(groupId, "."), "/")
	metaFiles := []string{}
	finalMetaPath := path.Join(fixedRoot, gPath, artifactId, MAVEN_METADATA_FILE)
	files.StoreFile(finalMetaPath, content, true)
	metaFiles = append(metaFiles, finalMetaPath)
	if digest {
		metaFiles = append(metaFiles, genAllDigestFiles(finalMetaPath)...)
	}
	return metaFiles, nil
}

func genAllDigestFiles(metaFilePath string) []string {
	md5Path := metaFilePath + ".md5"
	sha1Path := metaFilePath + ".sha1"
	sha256Path := metaFilePath + ".sha256"
	digestFiles := []string{}
	if genDigestFile(md5Path, metaFilePath, files.MD5) {
		digestFiles = append(digestFiles, md5Path)
	}
	if genDigestFile(sha1Path, metaFilePath, files.SHA1) {
		digestFiles = append(digestFiles, sha1Path)
	}
	if genDigestFile(sha256Path, metaFilePath, files.SHA256) {
		digestFiles = append(digestFiles, sha256Path)
	}
	return digestFiles
}

func genDigestFile(hashFilePath, metaFilePath string, hashType crypto.Hash) bool {
	digestContent := files.Digest(metaFilePath, hashType)
	if digestContent != "" {
		files.StoreFile(hashFilePath, digestContent, true)
	} else {
		logger.Warn(
			fmt.Sprintf("Error: Can not create digest file %s for %s because of some missing folders",
				hashFilePath, metaFilePath))
		return false
	}
	return true
}

func fixRoot(root string) string {
	slashRoot := strings.TrimSpace(root)
	if slashRoot == "" {
		slashRoot = "/"
	}
	return slashRoot
}

func wildcardMetadataPaths(paths []string) []string {
	newPaths := []string{}
	for _, p := range paths {
		if strings.HasSuffix(p, MAVEN_METADATA_FILE) ||
			strings.HasSuffix(p, MAVEN_ARCH_FILE) {
			newPaths = append(newPaths, p[:len(p)-len(".xml")]+".*")
		} else if filepath.Ext(p) == ".md5" ||
			filepath.Ext(p) == ".sha1" ||
			filepath.Ext(p) == ".sha128" ||
			filepath.Ext(p) == ".sha256" {
			continue
		} else {
			newPaths = append(newPaths, p)
		}
	}
	return newPaths
}

func getSuffix(pkgType string, conf config.CharonConfig) []string {
	if !util.IsBlankString(pkgType) {
		return conf.GetIgnoreSignatureSuffix(pkgType)
	}
	return []string{}
}

func isIgnored(fileName string, ignorePatterns []string) bool {
	for _, ignoreName := range STANDARD_GENERATED_IGNORES {
		if !util.IsBlankString(fileName) &&
			strings.HasPrefix(fileName, strings.TrimSpace(ignoreName)) {
			logger.Info(
				fmt.Sprintf("Ignoring standard generated Maven path: %s", fileName))
			return true
		}
	}
	if len(ignorePatterns) > 0 {
		for _, dirs := range ignorePatterns {
			if match, err := regexp.MatchString(dirs, fileName); match && err == nil {
				return true
			}
		}
	}
	return false
}

func hashDecorateMetadata(fPath, metadata string) []string {
	hashes := []string{}
	for _, hash := range []string{".md5", ".sha1", ".sha256"} {
		hashes = append(hashes, path.Join(fPath, metadata+hash))
	}
	return hashes
}

func extractTarball(repo, prefix, dir_ string) string {
	if files.FileOrDirExists(repo) {
		logger.Info(fmt.Sprintf("Extracting tarball: %s", repo))
		tmpRoot, err := os.MkdirTemp(dir_, fmt.Sprintf("charon-%s-*", prefix))
		if err != nil {
			panic(err)
		}
		err = archive.ExtractZipAll(repo, tmpRoot)
		if err != nil {
			panic(err)
		}
		return tmpRoot
	}
	panic(fmt.Errorf("error: archive %s does not exist", repo))
}

type scannedPaths struct {
	topLevel string
	mvnPaths []string
	poms     []string
	dirs     []string
}

func (s scannedPaths) String() string {
	var sb strings.Builder
	sb.WriteString("Scanned paths: \n")
	sb.WriteString(fmt.Sprintf("Top level:%s\n", s.topLevel))

	appendLines := func(lines []string) {
		for _, l := range lines {
			sb.WriteString(l + "\n")
		}
	}
	sb.WriteString("Maven paths:\n")
	appendLines(s.mvnPaths)
	sb.WriteString("Pom paths:\n")
	appendLines(s.poms)
	sb.WriteString("Dirs:\n")
	appendLines(s.dirs)

	return sb.String()
}

// scan for paths and filter out the ignored paths,
// and also collect poms for later metadata generation
func scanPaths(ignorePatterns []string, filesRoot, root string) scannedPaths {
	logger.Info(fmt.Sprintf("Scan %s to collect files", filesRoot))
	topLevel := root
	validMvnPaths := []string{}
	nonMvnPaths := []string{}
	ignoredPaths := []string{}
	validPoms := []string{}
	validDirs := []string{}
	changedDirs := make(map[string]bool)
	topFound := false
	filepath.WalkDir(filesRoot, func(p string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			changedDirs[p] = true
			if !topFound {
				curDir := path.Base(p)
				if curDir == topLevel {
					topLevel = p
					topFound = true
				}
				tempRoot := path.Join(filesRoot, topLevel)
				if strings.TrimSuffix(p, "/") == strings.TrimSuffix(tempRoot, "/") {
					topLevel = tempRoot
					topFound = true
				}
			}
		} else {
			fName := path.Base(p)
			if strings.Contains(p, topLevel) {
				// Let's wait to do the regex / pom examination until we
				// know we're inside a valid root directory.
				if isIgnored(fName, ignorePatterns) {
					ignoredPaths = append(ignoredPaths, p)
				} else {
					validMvnPaths = append(validMvnPaths, p)
					if filepath.Ext(fName) == ".pom" {
						validPoms = append(validPoms, p)
					}
				}
			} else {
				nonMvnPaths = append(nonMvnPaths, p)
			}
		}
		return nil
	})
	if len(nonMvnPaths) > 0 {
		tmpNonMvnPaths := []string{}
		for _, n := range nonMvnPaths {
			tmpNonMvnPaths = append(tmpNonMvnPaths, strings.ReplaceAll(n, filesRoot, ""))
		}
		nonMvnPaths = tmpNonMvnPaths
		logger.Info(
			fmt.Sprintf("These files are not in the specified root dir %s, so will be ignored: \n%s",
				root, nonMvnPaths))
	}
	trimmedTop := strings.TrimSpace(topLevel)
	if !topFound || trimmedTop == "" || trimmedTop == "/" {
		logger.Warn(
			fmt.Sprintf(
				"Warning: the root path %s does not exist in tarball, will use empty trailing prefix for the uploading",
				topLevel))
		topLevel = filesRoot
	} else {
		for c := range changedDirs {
			if strings.HasPrefix(c, topLevel) {
				validDirs = append(validDirs, c)
			}
		}
	}
	logger.Info("Files scanning done.\n")
	if len(ignorePatterns) > 0 {
		logger.Info(
			fmt.Sprintf("Ignored paths with ignore_patterns %s as below:\n%s\n",
				ignorePatterns, strings.Join(ignorePatterns, "\n")))
	}

	return scannedPaths{
		topLevel: topLevel,
		mvnPaths: validMvnPaths,
		poms:     validPoms,
		dirs:     validDirs,
	}
}

// Collect GAVs and generating maven-metadata.xml.
//
// As all valid poms has been stored in s3 bucket, what we should do here is:
//   - Scan and get the GA for the poms
//   - Search all poms in s3 based on the GA
//   - Use searched poms to generate maven-metadata to refresh
func generateMetadatas(s3 storage.S3Client, poms []string,
	bucket, prefix, root string) map[string][]string {
	gaMap := make(map[string]bool)
	logger.Debug(fmt.Sprintf("Valid poms: %s", poms))
	validGAVsMap := parseGAVs(poms, root)
	for g, avs := range validGAVsMap {
		for a := range avs {
			logger.Debug(fmt.Sprintf("G: %s, A: %s", g, a))
			gPath := strings.Join(strings.Split(g, "."), "/")
			gaMap[path.Join(gPath, a)] = true
		}
	}
	// Here we don't need to add original poms, because they
	// have already been uploaded to s3 before calling this function
	allPoms := []string{}
	metaFiles := make(map[string][]string)
	for p := range gaMap {
		// avoid some wrong prefix, like searching org/apache
		// but got org/apache-commons
		gaPrefix := p
		if !util.IsBlankString(prefix) {
			gaPrefix = path.Join(prefix, p)
		}
		if !strings.HasSuffix(p, "/") {
			gaPrefix += "/"
		}
		existedPoms, success := s3.GetFiles(bucket, gaPrefix, ".pom")
		if len(existedPoms) == 0 {
			if success {
				logger.Debug(
					fmt.Sprintf("No poms found in s3 bucket %s for GA path %s",
						bucket, p))
				metaFilesDeletion, ok := metaFiles[META_FILE_DEL_KEY]
				if !ok {
					metaFilesDeletion = []string{}
				}
				metaFilesDeletion = append(metaFilesDeletion, path.Join(p, MAVEN_METADATA_FILE))
				metaFilesDeletion = append(metaFilesDeletion, hashDecorateMetadata(p, MAVEN_METADATA_FILE)...)
				metaFiles[META_FILE_DEL_KEY] = metaFilesDeletion
			} else {
				logger.Warn(
					fmt.Sprintf(
						"An error happened when scanning remote artifacts under GA path %s", p))
				metaFailedPaths, ok := metaFiles[META_FILE_FAILED]
				if !ok {
					metaFailedPaths = []string{}
				}
				metaFailedPaths = append(metaFailedPaths, path.Join(p, MAVEN_METADATA_FILE))
				metaFailedPaths = append(metaFailedPaths, hashDecorateMetadata(p, MAVEN_METADATA_FILE)...)
				metaFiles[META_FILE_FAILED] = metaFailedPaths
			}
		} else {
			logger.Debug(
				fmt.Sprintf("Got poms in s3 bucket %s for GA path %s: %s", bucket, p, poms))
			unPrefixedPoms := existedPoms
			if !util.IsBlankString(prefix) {
				unPrefixedPoms = []string{}
				if !strings.HasSuffix(prefix, "/") {
					for _, pom := range existedPoms {
						unPrefixedPoms = append(unPrefixedPoms, strings.TrimPrefix(pom, prefix))
					}
				} else {
					for _, pom := range existedPoms {
						unPrefixedPoms = append(unPrefixedPoms, strings.TrimPrefix(pom, prefix+"/"))
					}
				}
			}
			allPoms = append(allPoms, unPrefixedPoms...)
		}
	}
	gavMap := parseGAVs(allPoms, "/")
	if len(gavMap) > 0 {
		metaFilesGen := []string{}
		for g, avs := range gavMap {
			for a, vers := range avs {
				metas, err := genMetaFile(g, a, vers, root, true)
				if err != nil {
					logger.Warn(
						fmt.Sprintf(
							"Failed to create or update metadata file for GA %s:%s, please check if aligned Maven GA is correct in your tarball.",
							g, a))
				} else {
					logger.Debug(fmt.Sprintf("Generated metadata file %s for %s:%s", metaFiles, g, a))
					metaFilesGen = append(metaFilesGen, metas...)
				}
			}
		}
		metaFiles[META_FILE_GEN_KEY] = metaFilesGen
	}
	return metaFiles
}

// Determine whether the local archive contains /archetype-catalog.xml
// in the repo contents.
//
// If so, determine whether the archetype-catalog.xml is already
// available in the bucket. Merge (or unmerge) these catalogs and
// return a boolean indicating whether the local file should be uploaded.
func generateUploadArchetypeCatalog(s3 *storage.S3Client,
	bucket, root, prefix string) bool {
	remote := MAVEN_ARCH_FILE
	if !util.IsBlankString(prefix) {
		remote = path.Join(prefix, MAVEN_ARCH_FILE)
	}
	local := path.Join(root, MAVEN_ARCH_FILE)
	//  As the local archetype will be overwrittern later, we must keep
	//  a cache of the original local for multi-targets support
	localBak := path.Join(root, MAVEN_ARCH_FILE+".charon.bak")
	if files.FileOrDirExists(local) && !files.FileOrDirExists(localBak) {
		content, err := files.ReadFile(local)
		if err != nil {
			logger.Warn("Can not open file: " + local)
		} else {
			files.StoreFile(localBak, content, true)
		}
	}

	// If there is no local catalog, this is a NO-OP
	if files.FileOrDirExists(localBak) {
		existed, err := s3.FileExistsInBucket(bucket, remote)
		if err != nil {
			logger.Error(
				"Error: Can not generate archtype-catalog.xml due to: " + err.Error())
			return false
		}
		if !existed {
			genAllDigestFiles(local)
			// If there is no catalog in the bucket, just upload what we have locally
			return true
		} else {
			content, err := files.ReadFile(local)
			if err != nil {
				logger.Warn(
					fmt.Sprintf("Failed to parse archetype-catalog.xml from local archive with root: %s "+
						"becuase of error: %s. SKIPPING invalid archetype processing.", root, err))
				return false
			}
			localArchetypes, err := parseArchetypes(content)
			if err != nil {
				logger.Warn(
					fmt.Sprintf("Failed to parse archetype-catalog.xml from local archive with root: %s. "+
						"SKIPPING invalid archetype processing.", root))
				return false
			}
			if len(localArchetypes) < 1 {
				logger.Warn("No archetypes found in local archetype-catalog.xml, " +
					"even though the file exists! Skipping.")
			} else {
				// Read the archetypes from the bucket so we can do a merge / un-merge
				remoteXml, err := s3.ReadFileContent(bucket, remote)
				if err != nil {
					logger.Warn(fmt.Sprintf("Failed to get archetype-catalog.xml from bucket: %s. "+
						"OVERWRITING bucket archetype-catalog.xml with the valid, local copy.", bucket))
					return true
				}
				remoteArchetypes, err := parseArchetypes(remoteXml)
				if err != nil {
					logger.Warn(fmt.Sprintf("Failed to get archetype-catalog.xml from bucket: %s. "+
						"OVERWRITING bucket archetype-catalog.xml with the valid, local copy.", bucket))
					return true
				}
				if len(remoteArchetypes) == 0 {
					genAllDigestFiles(local)
					// Nothing in the bucket. Just push what we have locally.
					return true
				} else {
					originalRemoteSize := len(remoteArchetypes)
					for _, la := range localArchetypes {
						// The cautious approach in this operation contradicts
						// assumptions we make for the rollback case.
						// That's because we should NEVER encounter a collision
						// on archetype GAV...they should belong with specific
						// product releases.
						// Still, we will WARN, not ERROR if we encounter this.
						if !slices.Contains(localArchetypes, la) {
							remoteArchetypes = append(remoteArchetypes, la)
						} else {
							logger.Warn(fmt.Sprintf("\n\n\nDUPLICATE ARCHETYPE: %s. "+
								"This makes rollback of the current release UNSAFE!\n\n\n", la))
						}
					}
					if len(remoteArchetypes) != originalRemoteSize {
						// If the number of archetypes in the version of
						// the file from the bucket has changed, we need
						// to regenerate the file and re-upload it.
						//
						// Re-render the result of our archetype merge /
						// un-merge to the local file, in preparation for
						// upload.
						arch := NewMavenArchetypeCatalog(remoteArchetypes)
						content, err = arch.GenerateMetaFileContent()
						if err != nil {
							logger.Error(fmt.Sprintf(
								"Error: Can not create file %s because of some missing folders", local))
							return false
						}
						files.StoreFile(local, content, true)
						genAllDigestFiles(local)
						return true
					}
				}
			}
		}
	}

	return false
}

func parseArchetypes(archXmlContent string) ([]ArchetypeRef, error) {
	archCatalog := MavenArchetypeCatalog{}
	err := xml.Unmarshal([]byte(archXmlContent), &archCatalog)
	if err != nil {
		logger.Error("Can not parse archetype-catalog file " + archXmlContent)
		return nil, err
	}
	return archCatalog.Archetypes, nil
}

func validateMaven(paths []string) ([]string, bool) {
	// Reminder: need to implement later
	logger.Debug(fmt.Sprintf("Need to validate mvn paths: %s", paths))
	return []string{}, true
}

func handleError(errMsgs []string) {
	// Reminder: will implement later
}

func versionCompare(ver1, ver2 string) int {
	xitems := strings.Split(ver1, ".")
	if strings.Contains(xitems[len(xitems)-1], "-") {
		xitems = append(xitems[0:len(xitems)-1], strings.Split(xitems[len(xitems)-1], "-")...)
	}
	yitems := strings.Split(ver2, ".")
	if strings.Contains(yitems[len(yitems)-1], "-") {
		yitems = append(yitems[0:len(yitems)-1], strings.Split(yitems[len(yitems)-1], "-")...)
	}
	big := max(len(xitems), len(yitems))
	for i := 0; i < big; i++ {
		if i >= len(xitems) {
			return -1
		}
		if i >= len(yitems) {
			return 1
		}
		xitem := xitems[i]
		yitem := yitems[i]
		if isInt(xitem) && !isInt(yitem) {
			return 1
		} else if !isInt(xitem) && isInt(yitem) {
			return -1
		} else if isInt(xitem) && isInt(yitem) {
			xitemInt, _ := strconv.Atoi(xitem)
			yitemInt, _ := strconv.Atoi(yitem)
			if xitemInt > yitemInt {
				return 1
			} else if xitemInt < yitemInt {
				return -1
			}
		} else {
			if xitem > yitem {
				return 1
			} else if xitem < yitem {
				return -1
			} else {
				continue
			}
		}
	}

	return 0
}

func archetypeRefCompare(arch1, arch2 ArchetypeRef) int {
	x := arch1.GroupId + ":" + arch1.ArtifactId
	y := arch2.GroupId + ":" + arch2.ArtifactId

	if x == y {
		return versionCompare(arch1.Version, arch2.Version)
	} else if x < y {
		return -1
	}
	return 1

}
