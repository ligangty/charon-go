package main

import (
	"fmt"
	"log/slog"
	"os"

	"org.commonjava/charon/pkg/storage"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func main() {
	s3Client, err := storage.NewS3Client("ronda", 0, false)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	// files, ok := s3Client.GetFiles("dev-maven-bucket", "ga/org", "")
	// if ok {
	// 	for _, file := range files {
	// 		fmt.Println(file)
	// 	}
	// }

	content, err := s3Client.ReadFileContent("dev-maven-bucket", "ga/org/apache/activemq/artemis-native/2.6.3.jbossorg-001/artemis-native-2.6.3.jbossorg-001.jar")
	if err == nil {
		fmt.Println(content)
	}

}
