

# Summary

There is a capability for Concourse to be implemented with a reverse proxy.  
However, we would like to harden this architecture by optionally only allowing authorised reverse proxies to access Concourse in this way using certificates to validate.  This can be done by implementing support for  mTLS within Concourse.
The mTLS protocol is defined in https://docs.oracle.com/cd/E19798-01/821-1841/bncbt/index.html and  RFC-8705 (at proposed stage) and a small example of the workings of the protocol can be found [here]


# Motivation

Our motivation for this is as follows:

By making the connection authenticate in both directions, it reduces the risk of a reverse proxy accidentally being configured to point to the wrong Concourse instance particularly in more complex networking environments.

By only allowing connections from one source, it reduces the attack surface of the Concourse instance and effectively locks the reverse proxy to the Concourse instance.  While this could also be achieved by use of firewalling, this allows the restriction to managed within the subsystem.


# Proposal
At present, Concourse supports TLS connectivity by specifying the --tls-cert and --tls-key parameters.  These are used by atc/atccmd/command.go to instantiate a TlsConfig object which is used to configure the listener.
Our proposal is to add a parameter 'tls-ca-cert' which will point to a file containing the certificate data that incoming connections will be validated against.
As such, there will be no end user visible changes except in environments where users have the option of connecting directly to Concourse as well as via a proxy.  In that case, a valid certificate will have to be provided by the client.

# Implementation
The component to be changed is atc/atccmd/command.go function tlsConfig which will be amended as follows:
<pre><code>
func (cmd *RunCommand) tlsConfig(logger lager.Logger, dbConn db.Conn) (*tls.Config, error) {
        var tlsConfig *tls.Config
        tlsConfig = atc.DefaultTLSConfig()

        if cmd.isTLSEnabled() {
                tlsLogger := logger.Session("tls-enabled")
               <b> if cmd.isMTLSEnabled() {
                        clientCACert, err := ioutil.ReadFile(string(cmd.TLSCaCert))
                        if err != nil {
                                return nil, err
                        }
                        clientCertPool := x509.NewCertPool()
                        clientCertPool.AppendCertsFromPEM(clientCACert)

                        tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
                        tlsConfig.ClientCAs = clientCertPool
                }
                </b>
                ...
 func (cmd *RunCommand) isMTLSEnabled() bool {
        return string(cmd.TLSCaCert) != ""
}
            
</pre>
A full implementation can be seen [here](https://github.com/nickhyoti/concourse-1/blob/48d34c191b259ebeade40a542e7a6e510f702997/atc/atccmd/command.go#L1321)

As well as making this change, the testing ... will also have to be amended:
testflight/suite_test.go - an additional environmental parameter CA_CERT to allow the ca-cert parameter to be set to 'fly' when self-signed certificates are in use.










# Open Questions

The mTLS exchange is defined in https://docs.oracle.com/cd/E19798-01/821-1841/bncbt/index.html and  RFC-8705  is at proposed status.  As such, while there is a risk that the RFC may not be finalised, the mechanism has been implemented in several environments particularly in B2B.
Do the web tests, in particular web/wats, have to pass with mTLS enabled?




