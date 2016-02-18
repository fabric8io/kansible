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

### Running GoSupervise
  
To try out running one of the example Ansible provisioned apps try the following:

* [Download a release](https://github.com/fabric8io/gosupervise/releases) and add `gosupervise` to your `$PATH` 
* Or [Build gosupervise](https://github.com/fabric8io/gosupervise#building) then add the `$PWD/bin` folder to your `$PATH` so that you can type in `gosupervise` on the command line

The following examples use these files:

* `inventory` is the Ansible inventory file used unless you specify the `--inventory` command line option
* `rc.yml` is the Replication Controller configuration used for the supervisor pods unless you specify the `--rc` command line option

 
#### for [fabric8-ansible-spring-boot](https://github.com/fabric8io/fabric8-ansible-spring-boot)

type the following to setup the VMs and provision things with Ansible

```
    git clone https://github.com/fabric8io/fabric8-ansible-spring-boot.git
    cd fabric8-ansible-spring-boot
    vagrant up
    ansible-playbook -i inventory provisioning/site.yml -vv
```
    
Now to setup the Replication Controller for the supervisors run the following, where `appservers` is the hosts from the inventory
    
```    
    gosupervise rc appservers
```      

The pods should now start up for each host in the inventory!


### [fabric8-ansible-hawtapp](https://github.com/fabric8io/fabric8-ansible-hawtapp)

type the following to setup the VMs and provision things with Ansible

```
    git clone https://github.com/fabric8io/fabric8-ansible-hawtapp.git
    cd fabric8-ansible-hawtapp
    vagrant up
    ansible-playbook -i inventory provisioning/site.yml -vv
```
    
Now to setup the Replication Controller for the supervisors run the following, where `appservers` is the hosts from the inventory

```    
    gosupervise rc appservers
```      

The pods should now start up for each host in the inventory!

To try using windows machines, replace `appservers` with `winboxes` in the above commands; assuming you have created the [Windows vagrant machine](https://github.com/fabric8io/fabric8-ansible-hawtapp/tree/master/windows) locally


### Checking the runtime status of the supervisors
 
To see which pods own which hosts run the following command:
 
```
    oc export rc hawtapp-demo | grep ansible.fabric8  | sort
```

Where `hawtapp-demo` is the name of the RC for the supervisors.

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

## Running pods locally

You can run `gosupervise rc ...` easily on a local build when working on the code. However to try out changes to the pod for `gosupervise pod ...` you can run that locally outside of docker with a small trick.

You must set the `HOSTNAME` environment variable to a valid pod name you wish to use.

```
    export HOSTNAME=fabric8-znuj5
```

e.g. the above uses the pod name for the current fabric8 console.

This lets you pretend to be different pods from the command line when trying it out locally. e.g. run the `gosupervise pod ...` command in 2 shells as different pods, provided the `HOSTNAME` values are diferent.

The pod name must be valid as `gosupervise pod ...` command will update the pod to annotate which host its chosen etc.

So to run the [above examples](#running-gosupervise) type the following:

for [fabric8-ansible-spring-boot](https://github.com/fabric8io/fabric8-ansible-spring-boot):
    
```    
    gosupervise pod -rc hawtapp-demo appservers /opt/cdi-camel-2.2.98-SNAPSHOT-app/bin/run.sh
```      

for [fabric8-ansible-hawtapp](https://github.com/fabric8io/fabric8-ansible-hawtapp):

```    
    gosupervise pod  -rc springboot-demo appservers /opt/springboot-camel-2.2.98-SNAPSHOT
```      


## License

Copyright 2016 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at <http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
