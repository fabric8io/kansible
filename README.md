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

* add the `$PWD/bin` folder to your `$PATH` so that you can type in `gosupervise` on the command line

The following examples use these files:

* `inventory` is the Ansible inventory file used unless you specify the `--inventory` command line option
* `rc.yml` is the Replication Controller configuration used for the supervisor pods unless you specify the `--rc` command line option

### [fabric8-ansible-hawtapp](https://github.com/fabric8io/fabric8-ansible-hawtapp)

type the following to setup the VMs and provision things with Ansible

```
    git clone git@github.com:fabric8io/fabric8-ansible-hawtapp.git
    cd fabric8-ansible-hawtapp
    vagrant up
    ansible-playbook -i inventory provisioning/site.yml -vv
```
    
Now to setup the Replication Controller for the supervisors run the following, where `appservers` is the hosts from the inventory

```    
    gosupervise rc appservers
```      

To run the supervisor pod locally on one of the hosts run:
    
```    
    gosupervise pod appservers /opt/cdi-camel-2.2.98-SNAPSHOT-app/bin/run.sh
```      

To try using windows machines, replace `appservers` with `winboxes` in the above commands; assuming you have created the [Windows vagrant machine](https://github.com/fabric8io/fabric8-ansible-hawtapp/tree/master/windows) locally
 
#### for [fabric8-ansible-spring-boot](https://github.com/fabric8io/fabric8-ansible-spring-boot)

type the following to setup the VMs and provision things with Ansible

```
    git clone git@github.com:fabric8io/fabric8-ansible-spring-boot.git
    cd fabric8-ansible-spring-boot
    vagrant up
    ansible-playbook -i inventory provisioning/site.yml -vv
```
    
Now to setup the Replication Controller for the supervisors run the following, where `appservers` is the hosts from the inventory
    
```    
    gosupervise rc appservers
```      

To run the supervisor pod locally on one of the hosts run:
    
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

### Simulating multiple pods

When working on the code outside of Kubernetes its useful to simulate running pods. To do this just set the `HOSTNAME` environment variable to the pod name you wish to use:

```
    export HOSTNAME=supervisor-znuj5
```

This lets you pretend to be different pods from the command line when trying it out locally. e.g. run the `gosupervise pod ...` command in 2 shells as different pods.

Note that supervise pod checks all pods are still running and un-allocates dead pods; so you might want to cheat and use an existing pod name for your `HOSTNAME` to test out how multiple pods grab hosts etc.


 
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
