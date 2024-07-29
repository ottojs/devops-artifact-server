# Otto.js DevOps Artifact Server

## Summary

Upload your build artifact files easily.  
Lightweight replacement for JFrog Artifactory, Sonatype Nexus, etc.

## Project Status

This code should be considered experimental and not production quality.  
Additionally, you should only host this on your internal network with proper access controls (firewalls, etc).

## Uploading

At this time, an `ACCESS_KEY` is not needed to upload. This prevents needing to copy the secret to pipelines.  
This could lead to supply chain attacks but be sure your combination of `[org+project+type]` is unique enough.

```bash
curl -s \
  -X PUT \
  -F 'meta={"organization":"acme","project":"example","type":"text"};type=application/json' \
  -F "file=@./myfile.txt" \
  http://[ADDRESS]/upload;
```

## Downloading

Download the latest file by providing your access key and the same upload metadata.

```bash
wget "https://[ADDRESS]/download?organization=acme&project=example&type=text&version=latest&access_key=ACCESS_KEY"
```

## Building

```bash
mkdir ./bin/;
go mod tidy && go build -o ./bin/artifact-server -ldflags="-s -w" ./cmd/server/...;
```

## Warning

Having other systems (reverse proxies like nginx, HAProxy, etc.) can cause issues with body size, response time, and more. If you're experiencing issues, please be sure to run only the binary file to see if things improve. If so, you may need to reconfigure other software in your setup.
