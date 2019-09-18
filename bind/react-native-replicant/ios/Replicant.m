#import "Replicant.h"
#import <Repm/Repm.h>

@implementation Replicant
RepmConnection* conn;
NSString* clientID;

RCT_EXPORT_MODULE()

RCT_EXPORT_METHOD(dispatch:(NSString *)method arguments:(nonnull NSString *)arguments resolve:(RCTPromiseResolveBlock)resolve reject:(RCTPromiseRejectBlock)reject)
{
  [self ensureClientID];
  [self ensureConnection];
  if (conn == nil) {
    reject(@"Replicant", @"Could not open database", nil);
    return;
  }
  
  NSError *err;
  NSLog(@"Replicant: Calling: %@ with arguments: %@", method, arguments);
  NSData* res = [conn dispatch:method
                          data:[arguments dataUsingEncoding:NSUTF8StringEncoding]
                         error:&err];
  if (err != nil) {
    reject(@"Replicant", [NSString stringWithFormat:@"Error dispatching %@", method], err);
    return;
  }
  resolve(@[[[NSString alloc] initWithData:res encoding:NSUTF8StringEncoding]]);
}

- (void)ensureConnection {
  @synchronized (self) {
    if (conn != nil) {
      return;
    }
    
    if (clientID == nil) {
      NSLog(@"Replicant: clientID is nil, cannot open database");
      return;
    }
    
    NSString* repDir = [self replicantDir];
    if (repDir == nil) {
      return;
    }
    
    NSError *err;
    conn = RepmOpen(repDir, clientID, @"", &err);
    if (err != nil) {
      NSLog(@"Replicant: Could not open database: %@", err);
      return;
    }
  }
}

- (void)ensureClientID {
  @synchronized (self) {
    if (clientID != nil) {
      return;
    }
    
    NSUserDefaults* defs = [NSUserDefaults standardUserDefaults];
    clientID = [defs stringForKey:@"clientID"];
    if (clientID != nil) {
      return;
    }
    
    CFUUIDRef uuid = CFUUIDCreate(NULL);
    NSString* uuidString = (NSString*)CFBridgingRelease(CFUUIDCreateString(NULL, uuid));
    CFRelease(uuid);
    
    [defs setValue:uuidString forKey:@"clientID"];
    BOOL ok = [defs synchronize];
    if (!ok) {
      NSLog(@"Replicant: could not save clientID to userdefaults");
      return;
    }
    
    clientID = uuidString;
  }
}

- (NSString*)replicantDir {
  NSFileManager* sharedFM = [NSFileManager defaultManager];
  NSArray* possibleURLs = [sharedFM URLsForDirectory:NSApplicationSupportDirectory
                                           inDomains:NSUserDomainMask];
  NSURL* appSupportDir = nil;
  NSURL* dataDir = nil;
  
  if ([possibleURLs count] < 1) {
    NSLog(@"Replicant: Could not location application support directory: %@", dataDir);
    return nil;
  }
  
  // Use the first directory (if multiple are returned)
  appSupportDir = [possibleURLs objectAtIndex:0];
  NSString* appBundleID = [[NSBundle mainBundle] bundleIdentifier];
  dataDir = [appSupportDir URLByAppendingPathComponent:appBundleID];
  dataDir = [dataDir URLByAppendingPathComponent:@"replicant"];
  
  NSError *err;
  [sharedFM createDirectoryAtPath:[dataDir path] withIntermediateDirectories:TRUE attributes:nil error:&err];
  if (err != nil) {
    NSLog(@"Replicant: Could not create data directory: %@", dataDir);
    return nil;
  }
  
  return [dataDir path];
}

@end
