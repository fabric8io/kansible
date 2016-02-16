# Go Supervise

This process supervises a running process on a remote machine.

## Building

* install [glide](https://github.com/Masterminds/glide#install)
* prepare the `vendor` folder by typing

```
    make bootstrap
    export GO15VENDOREXPERIMENT=1
```
    
* then to build the binary
    
```
    make build
```
    
* you can then run it via

```    
    ./bin/gosupervise
```

### Trying it out
  
To try out running one of the example anible provisioned apps try the following:

* add the `$PWD/bin` folder to your `$PATH`

### [fabric8-ansible-hawtapp](https://github.com/fabric8io/fabric8-ansible-hawtapp)

type the following to setup the VMs and provision things with Ansible

```
    git clone git@github.com:fabric8io/fabric8-ansible-hawtapp.git
    cd fabric8-ansible-hawtapp
    vagrant up
    ansible-playbook -i inventory provisioning/site.yml -vv
```
    
Now to run the supervisor on one of the hosts run:
    
```    
    gosupervise ansible --inventory inventory appservers /opt/cdi-camel-2.2.98-SNAPSHOT-app/bin/run.sh
```      
  
#### for [fabric8-ansible-spring-boot](https://github.com/fabric8io/fabric8-ansible-spring-boot)

type the following to setup the VMs and provision things with Ansible

```
    git clone git@github.com:fabric8io/fabric8-ansible-spring-boot.git
    cd fabric8-ansible-spring-boot
    vagrant up
    ansible-playbook -i inventory provisioning/site.yml -vv
```
    
Now to run the supervisor on one of the hosts run:
    
```    
    gosupervise ansible --inventory inventory appservers /opt/springboot-camel-2.2.98-SNAPSHOT
```      
  
## License

Copyright 2016 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at <http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
