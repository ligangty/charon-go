package main

import (
	"log/slog"
	"os"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func main() {
	// s3Client, err := storage.NewS3Client("ronda", 0, false)
	// if err != nil {
	// 	logger.Error(err.Error())
	// 	os.Exit(1)
	// }

	// files, ok := s3Client.GetFiles("dev-maven-bucket", "", "")
	// if ok {
	// 	for _, file := range files {
	// 		fmt.Println(file)
	// 	}
	// }

	// content, err := s3Client.ReadFileContent("dev-maven-bucket", "ga/org/apache/activemq/artemis-native/2.6.3.jbossorg-001/artemis-native-2.6.3.jbossorg-001.jar")
	// if err == nil {
	// 	fmt.Println(content)
	// }

	// content := s3Client.ListFolderContent("dev-maven-bucket", "ga/org/")
	// for _, c := range content {
	// 	fmt.Println(c)
	// }

	// ok, err := s3Client.FileExistsInBucket("dev-maven-bucket", "ga/org/index.html")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(ok)

	// err = s3Client.SimpleUploadFile("org/test/simpletest", "this is a test",
	// 	[2]string{"dev-maven-bucket", "ga"}, "plain/text", "", true)
	// if err == nil {
	// 	fmt.Println("Upload Successfully!")
	// } else {
	// 	fmt.Printf("Error: %s", err)
	// }
	// ok, _ := s3Client.FileExistsInBucket("dev-maven-bucket", "ga/org/test/simpletest")
	// fmt.Println(ok)
	// ok = s3Client.SimpleDeleteFile("org/test/simpletest", util.Target{Bucket: "dev-maven-bucket", Prefix: "ga"})
	// if ok {
	// 	fmt.Println("Delete Successfully!")
	// } else {
	// 	fmt.Println("Delete not successfully!")
	// }
	// ok, _ = s3Client.FileExistsInBucket("dev-maven-bucket", "ga/org/test/simpletest")
	// fmt.Println(ok)

}
