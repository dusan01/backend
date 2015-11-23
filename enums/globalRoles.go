package enums

var GlobalRoles = struct {
  Dummy,
  Guest,
  User,
  BronzeDonator,
  SilverDonator,
  GoldDonator,
  PlatinumDonator,
  TrialAmbassador,
  Ambassador,
  TrustedAmbassador,
  Admin,
  Server int
}{
  Dummy:             0,
  Guest:             1,
  User:              1,
  BronzeDonator:     50,
  SilverDonator:     51,
  GoldDonator:       52,
  PlatinumDonator:   53,
  TrialAmbassador:   400,
  Ambassador:        401,
  TrustedAmbassador: 402,
  Admin:             999,
  Server:            1000,
}
