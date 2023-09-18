buildDir=$(dirname "$0")
source $buildDir/lib/lib.sh

buildDir=$(dirname $(ireadlink "$0"))

mkdir -p $buildDir/certs
rm -f $buildDir/certs/*
openssl req -new -nodes -x509 -out $buildDir/certs/server.pem -keyout $buildDir/certs/server.key -days 36500 -subj "/C=US/ST=NYK/L=SP/O=P/OU=IT/CN=it.helloworld.club/emailAddress=it@helloworld.club"
