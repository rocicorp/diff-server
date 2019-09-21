Pod::Spec.new do |s|
  s.name             = 'replicant'
  s.version          = '0.0.1'
  s.summary          = 'iOS support code for the React Native bindings for Replicant'
  s.description      = <<-DESC
This pod contains native iOS code that supports the replicant-react-native npm package. It's not meant to be used independently.
                       DESC
  s.homepage         = 'https://replicate.to'
  s.license          = { :file => 'LICENSE' }
  s.author           = { 'Rocicorp' => 'info@roci.dev' }
  s.source           = { :path => 'ios/' }
  s.source_files = 'ios/*.{h,m}'
  s.public_header_files = 'ios/*.h'
  s.dependency 'React'
  s.vendored_frameworks = 'ios/Frameworks/Repm.framework'

  s.ios.deployment_target = '8.0'
end


