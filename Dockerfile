FROM fedora:30
MAINTAINER Henryk Konsek <hekonsek@gmail.com>

ADD biedatwitter /usr/bin/biedatwitter

CMD ["/usr/bin/biedatwitter"]