sudo apt-get install -y glances
sudo apt-get install -y nginx

# https://github.com/tkuchiki/alp
curl -sLO https://github.com/tkuchiki/alp/releases/download/v0.2.4/alp_linux_amd64.zip
unzip alp_linux_amd64.zip
sudo mv alp /usr/local/bin/alp


# https://github.com/KLab/myprofiler
wget https://github.com/KLab/myprofiler/releases/download/0.1/myprofiler.linux_amd64.tar.gz
tar xf myprofiler.linux_amd64.tar.gz
sudo mv myprofiler /usr/local/bin/

# nginx
sudo apt-get install -y build-essential \
    libpcre3 libpcre3-dev zlib1g-dev libssl-dev glances

PATH=/home/isucon/.local/go/bin:$PATH GOROOT=/home/isucon/.local/go GOPATH=/home/isucon/webapp/go go get -u -v github.com/cubicdaiya/nginx-build
nginx-build -d work \
    --sbin-path=/usr/sbin/nginx \
    --conf-path=/etc/nginx/nginx.conf \
    --with-http_gzip_static_module


# monitoring

sudo apt-get install -y glances iotop
