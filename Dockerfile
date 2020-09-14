FROM ubuntu:20.04
LABEL maintainer="540173107@qq.com"

# set timezome
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
RUN  sed -i s@/archive.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list
RUN  apt-get clean
RUN apt-get update
RUN apt-get install apt-utils -y
RUN apt-get install -f


RUN mkdir /pkage
RUN mkdir /data
RUN mkdir /gopath
RUN mkdir /gopath/src
RUN mkdir /gopath/bin
RUN mkdir /gopath/pkg

RUN apt-get install wget -y
RUN wget -O /pkage/go1.15.2.linux-amd64.tar.gz https://dl.google.com/go/go1.15.2.linux-amd64.tar.gz
RUN cd /pkage && tar -C /usr/local -zxvf go1.15.2.linux-amd64.tar.gz
RUN echo PATH="\$PATH:/usr/local/go/bin" >> /etc/profile
RUN echo "export GOROOT=/usr/local/go"  >> /etc/profile
RUN echo "export GOBIN=$GOROOT/bin"  >> /etc/profile
RUN echo "export GOPATH=/gopath"  >> /etc/profile
RUN echo "export GOPROXY=https://goproxy.io,direct" >> /etc/profile
RUN echo "source /etc/profile" >> /root/.bashrc
RUN wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
RUN apt install ./google-chrome-stable_current_amd64.deb -y
