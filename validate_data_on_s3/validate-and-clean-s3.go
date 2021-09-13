package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	envAWSRegion       = "AWS_REGION"
	envS3Bucket        = "S3_BUCKET_NAME"
	envS3Action        = "S3_ACTION"
	envS3Prefix        = "S3_PREFIX"
	envTestFile        = "TEST_FILE"
	envExpectedLogsLen = "EXPECTED_EVENTS_LEN"
)

type Message struct {
	Log string
}

func main() {
	os.Setenv(envAWSRegion, "us-west-2")
	os.Setenv(envS3Bucket, "fluent-bit-perf-test")
	os.Setenv(envS3Prefix, "flb-perf-test")
	os.Setenv(envTestFile, "/home/ec2-user/fluent-bit-performance-test/data/perf-test/input/test.log")
	os.Setenv(envS3Action, "validate")
	// os.Setenv(envS3Action, "clean")

	region := os.Getenv(envAWSRegion)
	if region == "" {
		exitErrorf("[TEST FAILURE] AWS Region required. Set the value for environment variable- %s", envAWSRegion)
	}

	bucket := os.Getenv(envS3Bucket)
	if bucket == "" {
		exitErrorf("[TEST FAILURE] Bucket name required. Set the value for environment variable- %s", envS3Bucket)
	}

	prefix := os.Getenv(envS3Prefix)
	if prefix == "" {
		exitErrorf("[TEST FAILURE] S3 object prefix required. Set the value for environment variable- %s", envS3Prefix)
	}

	testFile := os.Getenv(envTestFile)
	if testFile == "" {
		exitErrorf("[TEST FAILURE] test verfication file name required. Set the value for environment variable- %s", envTestFile)
	}

	// expectedEventsLen := os.Getenv(envExpectedLogsLen)
	// if expectedEventsLen == "" {
	// 	exitErrorf("[TEST FAILURE] number of expected log events required. Set the value for environment variable- %s", envExpectedLogsLen)
	// }
	// numEvents, convertionError := strconv.Atoi(expectedEventsLen)
	// if convertionError != nil {
	// 	exitErrorf("[TEST FAILURE] String to Int convertion Error for EXPECTED_EVENTS_LEN:", convertionError)
	// }

	s3Client, err := getS3Client(region)
	if err != nil {
		exitErrorf("[TEST FAILURE] Unable to create new S3 client: %v", err)
	}

	s3Action := os.Getenv(envS3Action)
	if s3Action == "validate" {
		// Validate the data on the s3 bucket
		getS3ObjectsResponses := getS3Objects(s3Client, bucket, prefix)
		validate(s3Client, getS3ObjectsResponses, bucket, testFile)
	} else {
		// Clean the s3 bucket-- delete all objects
		deleteS3Objects(s3Client, bucket, prefix)
	}
}

// Creates a new S3 Client
func getS3Client(region string) (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)

	if err != nil {
		return nil, err
	}

	return s3.New(sess), nil
}

// Returns all the objects from a S3 bucket with the given prefix
func getS3Objects(s3Client *s3.S3, bucket string, prefix string) []*s3.ListObjectsV2Output {
	index := 0
	var continuationToken *string

	var responseList []*s3.ListObjectsV2Output
	var input *s3.ListObjectsV2Input
	for {
		input = &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			ContinuationToken: continuationToken,
			Prefix:            aws.String(prefix),
		}

		response, err := s3Client.ListObjectsV2(input)

		if err != nil {
			exitErrorf("[TEST FAILURE] Error occured to get the objects from bucket: %q., %v", bucket, err)
		}

		responseList = append(responseList, response)
		index++

		if !aws.BoolValue(response.IsTruncated) {
			break
		}
		continuationToken = response.NextContinuationToken
	}

	return responseList
}

// Validates the log messages. Our log producer is designed to send 1000 integers [0 - 999].
// Both of the Kinesis Streams and Kinesis Firehose try to send each log maintaining the "at least once" policy.
// To validate, we need to make sure all the valid numbers [0 - 999] are stored at least once.
func validate(s3Client *s3.S3, responses []*s3.ListObjectsV2Output, bucket string, testFile string) {
	s3RecordCounter := 0
	s3ObjectCounter := 0
	recordFound := 0

	inputMap, err := readIdsFromFile(testFile)
	if err != nil {
		exitErrorf("[TEST FAILURE] Error to parse input file: %v", err)
	}
	totalInputRecord := len(inputMap)

	for _, response := range responses {
		for i := range response.Contents {
			input := &s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    response.Contents[i].Key,
			}
			obj := getS3Object(s3Client, input)
			s3ObjectCounter++

			dataByte, err := ioutil.ReadAll(obj.Body)
			if err != nil {
				exitErrorf("[TEST FAILURE] Error to parse GetObject response. %v", err)
			}

			data := strings.Split(string(dataByte), "\n")

			for _, d := range data {
				if d == "" {
					continue
				}

				var message Message

				decodeError := json.Unmarshal([]byte(d), &message)
				if decodeError != nil {
					exitErrorf("[TEST FAILURE] Json Unmarshal Error:", decodeError)
				}

				// fmt.Println(message)
				id := message.Log[:8]
				number, convertionError := strconv.Atoi(id)
				if convertionError != nil {
					exitErrorf("[TEST FAILURE] String to Int convertion Error:", convertionError)
				}

				if number < 0 {
					exitErrorf("[TEST FAILURE] Invalid number: %d found. Expected value in range (0 - %d)", number)
				}
				s3RecordCounter += 1

				if _, ok := inputMap[number]; ok {
					inputMap[number] = true
				}
			}

		}
	}

	for _, v := range inputMap {
		if v {
			recordFound++
		}
	}

	fmt.Println("Total input record: ", totalInputRecord)
	fmt.Println("Total object in S3: ", s3ObjectCounter)
	fmt.Println("Total record in S3: ", s3RecordCounter)
	fmt.Println("Duplicate records:", (s3RecordCounter - recordFound))
	if totalInputRecord == recordFound {
		fmt.Println("[TEST SUCCESSFULL] Found all the log records.")
	} else {
		exitErrorf("[TEST FAILURE] Validation Failed. Number of missing log records: %d", totalInputRecord-recordFound)
	}
}

// Retrieves an object from a S3 bucket
func getS3Object(s3Client *s3.S3, input *s3.GetObjectInput) *s3.GetObjectOutput {
	obj, err := s3Client.GetObject(input)

	if err != nil {
		exitErrorf("[TEST FAILURE] Error occured to get s3 object: %v", err)
	}

	return obj
}

// Delete all the objects with the given prefix from the specified S3 bucket
func deleteS3Objects(s3Client *s3.S3, bucket string, prefix string) {
	// Setup BatchDeleteIterator to iterate through a list of objects.
	iter := s3manager.NewDeleteListIterator(s3Client, &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	// Traverse the iterator deleting each object
	if err := s3manager.NewBatchDeleteWithClient(s3Client).Delete(aws.BackgroundContext(), iter); err != nil {
		exitErrorf("[CLEAN FAILURE] Unable to delete the objects from the bucket %q., %v", bucket, err)
	}

	fmt.Println("[CLEAN SUCCESSFUL] All the objects are deleted from the bucket:", bucket)
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func readIdsFromFile(fileName string) (map[int]bool, error) {
	inputMap := make(map[int]bool)

	f, err := os.Open(fileName)
	defer f.Close()
	if err != nil {
		return inputMap, err
	}

	rd := bufio.NewReader(f)

	for {
		line, err := rd.ReadString('\n')

		if err == io.EOF {
			return inputMap, nil
		}

		if err != nil {
			return inputMap, err
		}

		line = line[:8]
		number, convertionError := strconv.Atoi(line)
		if convertionError != nil {
			exitErrorf("[TEST FAILURE] String to Int convertion Error:", convertionError)
		}
		inputMap[number] = false
	}
}
