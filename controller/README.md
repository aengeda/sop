This is the original template of what a controller will be from controller-runtime:

```
type Controller struct {
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (c *Controller) Register(ctx context.Context, mgr manager.Manager) error {
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(c)
}
```
