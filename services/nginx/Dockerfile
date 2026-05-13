FROM nginx:alpine

COPY nginx.conf /etc/nginx/nginx.conf
COPY conf.d/ /etc/nginx/conf.d/


RUN rm -f /etc/nginx/conf.d/default.conf

EXPOSE 80 443
