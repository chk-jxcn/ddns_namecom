# ddns_namecom
DDNS like api for name.com
You can run it on vps and call url from openwrt to update record

It could also help to update TXT record if you use certbot.

## usage
```
ddns_namecom [--bindaddress address]
```
default address: 127.0.0.1:5553, you should always run it behind reverse http proxy or access it from local machine.

## Get arguments
example:
```
curl http://127.0.0.1:5553/update?username=[username]&token=[token]&domain=[domain]&answer=[ip]&updateall=1&updatetype=A&deletedup=1
```
arguments:
```
domain: domain to update
username: name.com username
token: name.com API token
answer: dns answer
type: record Type, default "A"
recreate: delete all record of this domain and create a new one.
updateall: update all record, but name.com don't allow same host keep same answer, only one record could be update.
deletedup: delete record if dupcation record exist.
deleteall: delete all record

```
```
Note: if Type is A record and answer not present, it will use ip from header[X-Real-IP] or header[X-FORWARDED-FOR] or remote peer ip
```
## license
None
