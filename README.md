# FluentBit Playbook
This guide will provide you a step by step guideline to setup and test different fluent-bit input/output plugins.

## Index


## Load test for tail (input) and kinesis_firehose (output) plugin
https://github.com/fluent/fluent-bit/issues/3917

ssh -i "raykeypair.pem" ec2-user@ec2-35-84-194-209.us-west-2.compute.amazonaws.com

docker run --rm -v $(pwd):/fluent-bit/etc/ -v $HOME:/home -e "HOME=/home" --mount type=bind,source=/home/ec2-user/fluent-bit-performance-test/data/perf-test/fluent-bit.conf,destination=/fluent-bit/etc/fluent-bit.conf,readonly --mount type=bind,source=/home/ec2-user/fluent-bit-performance-test/data/perf-test/,destination=/data/perf-test/ fluent/fluent-bit:1.8.3-debug


With Log Level Debug:
docker run —rm -v $(pwd):/fluent-bit/etc/ -v $HOME:/home -e "HOME=/home" -e="FLB_LOG_LEVEL=debug" --mount type=bind,source=/home/ec2-user/fluent-bit-performance-test/data/perf-test/fluent-bit.conf,destination=/fluent-bit/etc/fluent-bit.conf,readonly --mount type=bind,source=/home/ec2-user/fluent-bit-performance-test/data/perf-test/,destination=/data/perf-test/ fluent/fluent-bit:1.8.3-debug

sudo tcpdump -i eth0 -n -s 120 -w core-fluentbit-1.8.3v1.pcap not port 22
scp -i "raykeypair.pem" ec2-user@ec2-35-84-194-209.us-west-2.compute.amazonaws.com:/home/ec2-user/core-fluentbit-1.8.3v3.pcap .

data/perf-test/runtest.sh > data/perf-test/logFolder-fb/test.log


Run FluentBit default image: v1.6.8 

*Debug:*
docker run --rm -v $(pwd):/fluent-bit/etc/ -v $HOME:/home -e "HOME=/home" -e="FLB_LOG_LEVEL=debug" --mount type=bind,source=/home/ec2-user/fluent-bit-performance-test/data/perf-test/fluent-bit.conf,destination=/fluent-bit/etc/fluent-bit.conf,readonly --mount type=bind,source=/home/ec2-user/fluent-bit-performance-test/data/perf-test/,destination=/data/perf-test/ fluent/fluent-bit:1.6.8-debug

*Without debug enabled:*
docker run --rm -v $(pwd):/fluent-bit/etc/ -v $HOME:/home -e "HOME=/home" --mount type=bind,source=/home/ec2-user/fluent-bit-performance-test/data/perf-test/fluent-bit.conf,destination=/fluent-bit/etc/fluent-bit.conf,readonly --mount type=bind,source=/home/ec2-user/fluent-bit-performance-test/data/perf-test/,destination=/data/perf-test/ fluent/fluent-bit:1.6.8-debug

Data generator:
[ec2-user@ip fluent-bit-performance-test]$ data/perf-test/runtest.sh > data/perf-test/logFolder-fb/test.log


Git Patch/ Diff:

git diff master aws_http_buffer > /home/ec2-user/aws_http_buffer.patch
git diff branch_1 branch_2 > output_file.patch
