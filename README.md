# Go Supervise

This process supervises a running process on a remote machine.

## Building

* install [glide](https://github.com/Masterminds/glide#install)
* prepare the `vendor` folder by typing

    make bootstrap
    
* then to build the binary
    
    make build
    
* you can then run it via
    
    ./bin/gosupervise

  
To try out running one of the example anible provisioned apps try the following

### for [fabric8-ansible-hawtapp](https://github.com/fabric8io/fabric8-ansible-hawtapp)
 
    ./bin/gosupervise  --host 10.10.3.20 --user vagrant --privatekey $FOO/fabric8-ansible-hawtapp/.vagrant/machines/app1/virtualbox/private_key --command "/opt/cdi-camel-2.2.98-SNAPSHOT-app/bin/run.sh" run

where `$FOO` points to the folder you cloned the ansible git repo
  
### for [fabric8-ansible-spring-boot](https://github.com/fabric8io/fabric8-ansible-spring-boot)

    ./bin/gosupervise  --host 10.10.2.20 --user vagrant --privatekey $FOO/fabric8-ansible-spring-boot/.vagrant/machines/app1/virtualbox/private_key --command "/opt/springboot-camel-2.2.98-SNAPSHOT" run

where `$FOO` points to the folder you cloned the ansible git repo  
  
## License

Copyright 2016 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at <http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
