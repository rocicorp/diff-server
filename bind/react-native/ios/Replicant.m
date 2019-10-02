#import "Replicant.h"
#import <Repm/Repm.h>

@implementation Replicant

  BOOL initialized = FALSE;
  NSString* replicantDir();

- (dispatch_queue_t)methodQueue
{
  return dispatch_queue_create("dev.roci.Replicant", NULL);
}

RCT_EXPORT_MODULE()

RCT_EXPORT_METHOD(dispatch:(NSString*)dbName method:(NSString *)method arguments:(nonnull NSString *)arguments resolve:(RCTPromiseResolveBlock)resolve reject:(RCTPromiseRejectBlock)reject)
{
  NSLog(@"Replicant: Calling: %@ with arguments: %@", method, arguments);
  if (!initialized) {
    RepmInit(replicantDir(), @"");
    initialized = true;
  }

  __block NSError *err;
  if ([method isEqualToString:@"sync"]) {
    dispatch_async(dispatch_get_global_queue( DISPATCH_QUEUE_PRIORITY_DEFAULT, 0), ^(void){
      NSData* res = [self request:dbName method:method args:arguments error:&err];
      dispatch_async(dispatch_get_main_queue(), ^(void){
        [self respond:method err:err result:res resolve:resolve reject:reject];
      });
    });
    return;
  }
  
  NSData *res = [self request:dbName method:method args:arguments error:&err];
  [self respond:method err:err result:res resolve:resolve reject:reject];
}

- (NSData*)request:(NSString*)dbName method:(NSString*)method args:(NSString*)args error:(NSError**)error {
  return RepmDispatch(dbName, method, [args dataUsingEncoding:NSUTF8StringEncoding], error);
}

- (void)respond:(NSString*)method err:(NSError*)err result:(NSData*)result resolve:(RCTPromiseResolveBlock)resolve reject:(RCTPromiseRejectBlock)reject {
  if (err != nil) {
    reject(@"Replicant", [NSString stringWithFormat:@"Error dispatching %@: %@", method, [err localizedDescription]], err);
    return;
  }
  resolve(@[[[NSString alloc] initWithData:result encoding:NSUTF8StringEncoding]]);
}

// TODO: Share with the same code in Flutter.
NSString* replicantDir() {
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

  NSError* err;
  [sharedFM createDirectoryAtPath:[dataDir path] withIntermediateDirectories:TRUE attributes:nil error:&err];
  if (err != nil) {
    NSLog(@"Replicant: Could not create data directory: %@", dataDir);
    return nil;
  }

  return [dataDir path];
}

@end
