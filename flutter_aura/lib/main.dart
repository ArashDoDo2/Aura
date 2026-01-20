import 'package:flutter/material.dart';
import 'aura_bridge.dart';

void main() {
  runApp(const AuraApp());
}

class AuraApp extends StatelessWidget {
  const AuraApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Aura DNS Tunnel',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple),
        useMaterial3: true,
      ),
      home: const AuraHomePage(),
    );
  }
}

class AuraHomePage extends StatefulWidget {
  const AuraHomePage({super.key});

  @override
  State<AuraHomePage> createState() => _AuraHomePageState();
}

class _AuraHomePageState extends State<AuraHomePage> {
  final TextEditingController _dnsController = TextEditingController();
  final TextEditingController _domainController = TextEditingController();
  
  bool _isConnected = false;
  bool _isLoading = false;
  String _statusMessage = 'Disconnected';

  @override
  void initState() {
    super.initState();
    // Default values
    _dnsController.text = ''; // Empty = system DNS
    _domainController.text = 'tunnel.example.com.';
  }

  Future<void> _toggleVpn() async {
    setState(() {
      _isLoading = true;
      _statusMessage = _isConnected ? 'Stopping...' : 'Starting...';
    });

    try {
      if (_isConnected) {
        AuraBridge.stopAura();
        setState(() {
          _isConnected = false;
          _statusMessage = 'Disconnected';
        });
        _showMessage('VPN Stopped', Colors.orange);
      } else {
        final dns = _dnsController.text.trim();
        final domain = _domainController.text.trim();
        
        if (domain.isEmpty) {
          _showMessage('Domain cannot be empty', Colors.red);
          setState(() {
            _isLoading = false;
            _statusMessage = 'Error: Domain required';
          });
          return;
        }

        AuraBridge.startAura(dns, domain);
        setState(() {
          _isConnected = true;
          _statusMessage = dns.isEmpty 
              ? 'Connected (System DNS)' 
              : 'Connected ($dns)';
        });
        _showMessage('VPN Started', Colors.green);
      }
    } catch (e) {
      _showMessage('Error: $e', Colors.red);
      setState(() {
        _isConnected = false;
        _statusMessage = 'Error: $e';
      });
    } finally {
      setState(() {
        _isLoading = false;
      });
    }
  }

  void _showMessage(String message, Color color) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: color,
        duration: const Duration(seconds: 2),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
        title: const Text('Aura DNS Tunnel'),
      ),
      body: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            // Status indicator
            Container(
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: _isConnected ? Colors.green.shade100 : Colors.grey.shade200,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Column(
                children: [
                  Icon(
                    _isConnected ? Icons.vpn_lock : Icons.vpn_lock_outlined,
                    size: 64,
                    color: _isConnected ? Colors.green : Colors.grey,
                  ),
                  const SizedBox(height: 12),
                  Text(
                    _statusMessage,
                    style: TextStyle(
                      fontSize: 18,
                      fontWeight: FontWeight.bold,
                      color: _isConnected ? Colors.green.shade900 : Colors.grey.shade700,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 32),
            
            // DNS Server input
            TextField(
              controller: _dnsController,
              enabled: !_isConnected && !_isLoading,
              decoration: InputDecoration(
                labelText: 'DNS Server (optional)',
                hintText: 'Empty = System DNS',
                helperText: 'Leave empty to use device DNS',
                border: const OutlineInputBorder(),
                prefixIcon: const Icon(Icons.dns),
              ),
            ),
            const SizedBox(height: 16),
            
            // Domain input
            TextField(
              controller: _domainController,
              enabled: !_isConnected && !_isLoading,
              decoration: const InputDecoration(
                labelText: 'Domain *',
                hintText: 'tunnel.example.com.',
                helperText: 'Must end with dot (.)',
                border: OutlineInputBorder(),
                prefixIcon: Icon(Icons.language),
              ),
            ),
            const SizedBox(height: 32),
            
            // Connect/Disconnect button
            SizedBox(
              width: double.infinity,
              height: 56,
              child: ElevatedButton(
                onPressed: _isLoading ? null : _toggleVpn,
                style: ElevatedButton.styleFrom(
                  backgroundColor: _isConnected ? Colors.red : Colors.green,
                  foregroundColor: Colors.white,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12),
                  ),
                ),
                child: _isLoading
                    ? const CircularProgressIndicator(color: Colors.white)
                    : Text(
                        _isConnected ? 'DISCONNECT' : 'CONNECT',
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
              ),
            ),
            
            const SizedBox(height: 24),
            
            // Info card
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: Colors.blue.shade50,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: Colors.blue.shade200),
              ),
              child: const Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Icon(Icons.info_outline, color: Colors.blue),
                      SizedBox(width: 8),
                      Text(
                        'About Aura',
                        style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                          color: Colors.blue,
                        ),
                      ),
                    ],
                  ),
                  SizedBox(height: 8),
                  Text(
                    '• DNS-based tunneling for WhatsApp text messages\n'
                    '• Port 5222 only (no media/voice)\n'
                    '• SOCKS5 proxy on localhost:1080\n'
                    '• Leave DNS empty for automatic detection',
                    style: TextStyle(fontSize: 14, height: 1.5),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  @override
  void dispose() {
    _dnsController.dispose();
    _domainController.dispose();
    super.dispose();
  }
}
