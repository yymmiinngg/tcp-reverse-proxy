buildDir=$(dirname "$0")
source $buildDir/lib/lib.sh

buildDir=$(dirname $(ireadlink "$0"))

echo "Init arguments..."
distDir=$buildDir/dist
mainDir=$buildDir/..
cd $mainDir

GOOS=$1
GOARCH=$2

if [ "$GOOS" == "" ]; then
     GOOS=linux
fi
if [ "$GOARCH" == "" ]; then
     GOARCH=amd64
fi

printf "  GOOS=%s # { linux | windows | darwin }\n" "$GOOS"
printf "  GOARCH=%s # { amd64 | 386 | arm | arm64 }\n" "$GOARCH"

echo "Building..."
appName=tcprp
appVersion=$(git describe --tags 2>/dev/null || date +"%Y%m%d%H%M%S")
appPlatform=$GOOS-$GOARCH
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

distName=$appName-$appPlatform-$appVersion

echo "Clear files..."
rm -rf $distDir/$distName
mkdir -p $distDir/$distName

echo "Copy files..."
mv $appName $distDir/$distName
cp $buildDir/lib/tcprps.script $distDir/$distName
cp $buildDir/lib/tcprps.conf $distDir/$distName
cp $buildDir/lib/tcprps.sh $distDir/$distName
cp $buildDir/lib/lib.sh $distDir/$distName

chmod +x $distDir/$distName/$appName
chmod +x $distDir/$distName/tcprps.sh

curDir=`pwd`
cd $distDir
zip -r $distName.zip $distName
cd $curDir

echo "done"
