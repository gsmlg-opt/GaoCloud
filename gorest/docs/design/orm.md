# Features
1. Primitive types to postgresql data type
- int, uint, string, bool
- time.Time, net.IP, net.IPNet
- []int, []string, []net.IP, []net.IPNet
1. Resource integrity
1. Resource relationship management

# Convensions
1. Table name is the snake format of the resource go struct name with "gr_" as 
   prefix
1. Id is unique with string type, when not specified will be generated use uuid.
1. use `pk` tag to speicify primiary key
1. use `uk` tag to include several composite attribute as unique constraint
1. `Get` and `Fill` interface returns slice of resource pointer
1. Resource create time is initialized during insert

# Thread Safety
1. `store.Begin` which used to get `Transaction` is thread safe
1. All the operation on one `Transaction` isn't thread safe

# Embed struct 
Each resource has `ResourceBase` as the first embed struct, during insertion, 
gorest DB module will automaticaly extract the `id` and `create_time` value , 
and during resource fetch, the related value will set into `ResourceBase`. 

# Relationship between resource
1. Use
```go
type View struct {
    ResourceBase 

    Name string 
    Acl string `db:"referto"`
}

type Acl struct {
    ResourceBase 

    Ips []net.IP
}
```
- `Acl` in `View` is valid acl id
- Used acl couldn't be deleted
1. One to many ownership  
```go
type View struct {
    ResourceBase

    Name string 
    Acl string `db:"referto"`
}

type Zone struct {
    ResourceBase

    View string `db:"ownby"`
}
```
- `View` in `Zone` is valid view id
- When view delete, all the zone owned by that view will be deleted too

1. Many to many ownership
```go
type View struct {
    ResourceBase

    Name string 
    Acl string `db:"referto"`
}

type User struct {
    ResourceBase

    Email string `db:"uk"`
}

type UserView struct {
    ResourceBase 

    User string `db:"ownby"`
    View string `db:"referto"`
}

views, err := tx.GetOwned("user", "user_id", "view") 
```
1. The relationship is a seperate table which means a seperate go struct
1. Delete owner will delete the relationship but not the resource being owned
1. The owned resource cann't be deleted once it is owned by any owner.
