sudo apt-get install -y glances
sudo apt-get install -y nginx

# https://github.com/tkuchiki/alp
curl -sLO https://github.com/tkuchiki/alp/releases/download/v0.2.4/alp_linux_amd64.zip
unzip alp_linux_amd64.zip
sudo mv alp /usr/local/bin/alp
