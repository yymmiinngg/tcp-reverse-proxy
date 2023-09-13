
buildDir=$(dirname "$0")
source $buildDir/lib/lib.sh

buildDir=$(dirname $(ireadlink "$0"))

distDir=$buildDir/dist
mainDir=$buildDir/..
cd $mainDir

# GOOS=linux
# GOARCH=amd64
appName=tcpt
appVersion=$(git describe --tags 2>/dev/null || date +"%Y%m%d%H%M%S")
appPlatform=$GOOS/$GOARCH
appBuildTime=$(date)
appGoVersion=$(go version)
CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
    go build -buildvcs=false \
    -o $appName \
    -ldflags "-X 'main.Name=$appName' \
    -X 'main.Version=$appVersion' \
    -X 'main.Platform=$appPlatform' \
    -X 'main.BuildTime=$appBuildTime' \
    -X 'main.GoVersion=$appGoVersion'"

rm -rf $distDir
mkdir -p $distDir

mv $appName $distDir
cp $buildDir/lib/tcpt.script $distDir
cp $buildDir/lib/singleton-run.sh $distDir
cp $buildDir/lib/lib.sh $distDir