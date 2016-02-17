# Go Supervise

This process supervises a running process on a remote machine.

### Configuring gosupervise

The best way to configure if you want to connect via SSH for unix machines or WinRM for windows machines is via the Ansible Inventory.

By default SSH is used on port 22 unless you specify `ansible_port` in the inventory or specify `--port` on the command line.

You can configure Windows machines using the `winrm=true` property in the inventory:


```yaml
[winboxes]
windows1 ansible_ssh_host=localhost ansible_port=5985 ansible_ssh_user=foo ansible_ssh_pass=somepasswd! winrm=true

[unixes]
app1 ansible_ssh_host=10.10.3.20 ansible_ssh_user=vagrant ansible_ssh_private_key_file=.vagrant/machines/app1/virtualbox/private_key
app2 ansible_ssh_host=10.10.3.21 ansible_ssh_user=vagrant ansible_ssh_private_key_file=.vagrant/machines/app2/virtualbox/private_key
```

You can also enable WinRM via the `--winrm` command line flag: 

```
export GOSUPERVISE_WINRM=true
gosupervise pod --winrm somehosts somecommand

```

or by setting the environment variable `GOSUPERVISE_WINRM` which is a little easier to configure on the RC YAML:

```
export GOSUPERVISE_WINRM=true
gosupervise pod somehosts somecommand

```


### Trying it out
  
To try out running one of the example Ansible provisioned apps try the following:

* add the `$PWD/bin` folder to your `$PATH`
* run the following command:

```
    export HOSTNAME=supervisor-znuj5
```
    
which gives the current shell a pod name. 

Note that the following examples cheat a little in that they use the Replication Controller called `fabric8` for now to store the ownership of pods -> hosts. When we create the RC for the supervisors then we should be using that RC instead ;)

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
    gosupervise pod appservers /opt/cdi-camel-2.2.98-SNAPSHOT-app/bin/run.sh
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
    gosupervise pod appservers /opt/springboot-camel-2.2.98-SNAPSHOT
```      

### Checking the runtime status of the supervisors
 
To see which pods own which hosts run the following command:
 
```
    oc export rc fabric8 | grep ansible.fabric8  | sort
```

Where `fabric8` is the name of the RC for the supervisors. (`fabric8` is a hack to reuse the fabric8 console for now until we actually make the RC ;).

The output is of the format:

```
    pod.ansible.fabric8.io/app1: supervisor-znuj5
    pod.ansible.fabric8.io/app2: supervisor-1same
```

Where the output is of the form ` pod.ansible.fabric8.io/$HOSTNAME: $PODNAME`

 
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

## License

Copyright 2016 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at <http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
